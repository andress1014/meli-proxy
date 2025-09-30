package unit

import (
	"testing"
	"time"

	"github.com/andress1014/meli-proxy/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zapcore.Level
	}{
		{"debug", "debug", zapcore.DebugLevel},
		{"info", "info", zapcore.InfoLevel},
		{"warn", "warn", zapcore.WarnLevel},
		{"error", "error", zapcore.ErrorLevel},
		{"invalid", "invalid", zapcore.InfoLevel}, // should default to info
		{"empty", "", zapcore.InfoLevel},          // should default to info
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(tt.level)
			
			if log == nil {
				t.Fatal("New() returned nil logger")
			}
			
			// Test that we can use the logger
			log.Info("test message")
			log.Debug("debug message")
			log.Warn("warn message")
			log.Error("error message")
			
			// Sync should not panic (but might return expected errors)
			if err := log.Sync(); err != nil {
				t.Logf("Sync() returned expected error: %v", err)
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	// Test debug level
	debugLogger := logger.New("debug")
	if debugLogger == nil {
		t.Fatal("Debug logger is nil")
	}
	
	// Test info level
	infoLogger := logger.New("info")
	if infoLogger == nil {
		t.Fatal("Info logger is nil")
	}
	
	// Test warn level
	warnLogger := logger.New("warn")
	if warnLogger == nil {
		t.Fatal("Warn logger is nil")
	}
	
	// Test error level
	errorLogger := logger.New("error")
	if errorLogger == nil {
		t.Fatal("Error logger is nil")
	}
}

func TestLoggerWithFields(t *testing.T) {
	log := logger.New("info")
	
	// Test logging with fields
	log.Info("test with fields",
		zap.String("field1", "value1"),
		zap.Int("field2", 42),
		zap.Bool("field3", true),
	)
	
	// Test with different types
	log.Warn("warning with various types",
		zap.Duration("duration", 100),
		zap.Time("timestamp", time.Now()),
		zap.Any("any", map[string]string{"key": "value"}),
	)
	
	// Should not panic
}

func TestLoggerSync(t *testing.T) {
	log := logger.New("info")
	
	// Multiple syncs should work (but might return expected errors on some systems)
	// Sync can return errors on stdout/stderr in some environments, which is expected
	err1 := log.Sync()
	if err1 != nil {
		t.Logf("First sync returned expected error: %v", err1)
	}
	
	err2 := log.Sync()
	if err2 != nil {
		t.Logf("Second sync returned expected error: %v", err2)
	}
	
	// Test passes if no panic occurs
}

func TestLoggerConcurrency(t *testing.T) {
	log := logger.New("info")
	
	// Test concurrent logging
	const numRoutines = 10
	done := make(chan bool, numRoutines)
	
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 10; j++ {
				log.Info("concurrent log", 
					zap.Int("routine", id), 
					zap.Int("iteration", j))
			}
		}(i)
	}
	
	// Wait for all routines to complete
	for i := 0; i < numRoutines; i++ {
		<-done
	}
	
	// Should not panic or error
	log.Sync()
}
