.PHONY: help build run test test-unit test-integration test-coverage test-bench test-race clean docker-build docker-run docker-stop dev stop logs fmt lint deps

# Variables
BINARY_NAME=meli-proxy
DOCKER_IMAGE=meli-proxy-optimized
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Comando por defecto
help:
	@echo ""
	@echo "🚀 MELI-PROXY - Comandos Disponibles"
	@echo "===================================="
	@echo ""
	@echo "📦 Build & Run:"
	@echo "  make build          - Compilar aplicación"
	@echo "  make run            - Ejecutar aplicación"
	@echo "  make clean          - Limpiar archivos generados"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test           - Ejecutar todos los tests"
	@echo "  make test-unit      - Tests unitarios solamente"
	@echo "  make test-integration - Tests de integración"
	@echo "  make test-coverage  - Tests con reporte de coverage"
	@echo "  make test-bench     - Ejecutar benchmarks"
	@echo "  make test-race      - Tests con race detection"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-build   - Build imagen Docker"
	@echo "  make docker-run     - Ejecutar con Docker Compose"
	@echo "  make docker-stop    - Detener contenedores"
	@echo ""
	@echo "🔧 Desarrollo:"
	@echo "  make deps           - Instalar/actualizar dependencias"
	@echo "  make fmt            - Formatear código"
	@echo "  make lint           - Ejecutar linter"
	@echo "  make dev            - Modo desarrollo"
	@echo ""

# Instalar/actualizar dependencias
deps:
	@echo "📦 Instalando dependencias..."
	go mod download
	go mod tidy
	@echo "✅ Dependencias actualizadas"

# Build the application
build: deps fmt
	@echo "🔨 Compilando $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) ./cmd/proxy
	@echo "✅ Build completado: bin/$(BINARY_NAME)"

# Run the application locally
run: build
	@echo "🚀 Ejecutando $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

# Tests unitarios solamente
test-unit:
	@echo "🧪 Ejecutando tests unitarios..."
	go test -v ./tests/unit/...
	@echo "✅ Tests unitarios completados"

# Tests de integración solamente
test-integration:
	@echo "🔗 Ejecutando tests de integración..."
	go test -v ./tests/integration/... -timeout=60s
	@echo "✅ Tests de integración completados"

# Tests con race detection
test-race:
	@echo "🏃 Ejecutando tests con race detection..."
	go test -race -v ./tests/...
	@echo "✅ Race detection tests completados"

# Tests completos
test: test-unit test-integration
	@echo "🎉 Todos los tests completados"

# Tests con coverage
test-coverage: deps
	@echo "📊 Ejecutando tests con coverage..."
	go test -coverprofile=$(COVERAGE_FILE) ./tests/...
	go tool cover -func=$(COVERAGE_FILE)
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "✅ Coverage report generado: $(COVERAGE_HTML)"

# Benchmarks
test-bench:
	@echo "🚀 Ejecutando benchmarks..."
	go test -bench=. -benchmem ./...
	@echo "✅ Benchmarks completados"

# Clean build artifacts
clean:
	@echo "🧹 Limpiando archivos generados..."
	rm -rf bin/
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	go clean -cache
	@echo "✅ Limpieza completada"

# Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

# Run with Docker Compose
docker-run: docker-build
	@echo "🐳 Iniciando servicios..."
	docker-compose up -d
	@echo "✅ Servicios iniciados"
	@echo "🌐 Proxy: http://localhost:8080"
	@echo "📊 Grafana: http://localhost:3000 (admin/admin)"

# Development mode (with hot reload)
dev:
	docker-compose up --build -d
	docker-compose logs -f meli-proxy

# Stop all services  
stop:
	@echo "🛑 Deteniendo contenedores..."
	docker-compose down
	@echo "✅ Contenedores detenidos"

docker-stop: stop

# View logs
logs:
	docker-compose logs -f

# Format code
fmt:
	@echo "🎨 Formateando código..."
	go fmt ./...
	@echo "✅ Código formateado"

# Lint code
lint:
	@echo "🔍 Ejecutando linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint no instalado, usando go vet..."; \
		go vet ./...; \
	fi
	@echo "✅ Linting completado"

# Run only Redis for development
redis:
	@echo "🔴 Iniciando Redis para desarrollo..."
	docker-compose up redis -d
	@echo "✅ Redis iniciado en puerto 6379"

# Load test with curl
load-test:
	@echo "Running basic load test..."
	@for i in {1..100}; do \
		curl -s http://localhost:8080/categories/MLA1234 > /dev/null & \
	done; \
	wait
	@echo "Load test completed"

# Check metrics
metrics:
	curl -s http://localhost:9090/metrics | grep meli_proxy

# Check health
health:
	curl -s http://localhost:8080/health | jq .
