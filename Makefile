.PHONY: build run test clean docker-build docker-run dev stop logs

# Variables
BINARY_NAME=meli-proxy
DOCKER_IMAGE=meli-proxy:latest

# Build the application
build:
	go build -o bin/$(BINARY_NAME) ./cmd/proxy

# Run the application locally
run:
	go run ./cmd/proxy

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE) .

# Run with Docker Compose
docker-run:
	docker-compose up --build

# Development mode (with hot reload)
dev:
	docker-compose up --build -d
	docker-compose logs -f meli-proxy

# Stop all services
stop:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run only Redis for development
redis:
	docker-compose up redis -d

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
