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
	@echo "  make docker-run-logs - Ejecutar con Loki + Grafana logs"
	@echo "  make docker-stop    - Detener contenedores"
	@echo "  make docker-stop-logs - Detener stack con logs"
	@echo ""
	@echo "🔧 Desarrollo:"
	@echo "  make deps           - Instalar/actualizar dependencias"
	@echo "  make fmt            - Formatear código"
	@echo "  make lint           - Ejecutar linter"
	@echo "  make dev            - Modo desarrollo"
	@echo ""
	@echo "📋 Logs & Monitoreo:"
	@echo "  make logs           - Ver logs de contenedores"
	@echo "  make logs-proxy     - Ver logs solo de proxies"
	@echo "  make open-grafana   - Abrir Grafana en navegador"
	@echo "  make test-loki      - Verificar API de Loki"
	@echo ""
	@echo "🚀 Deployment (Producción):"
	@echo "  make deploy-server  - Despliegue completo en servidor"
	@echo "  make setup-server   - Configurar servidor (primera vez)"
	@echo "  make deploy-app     - Actualizar solo la aplicación"
	@echo "  make check-server   - Verificar salud del servidor"
	@echo "  make server-logs    - Ver logs del servidor"
	@echo "  make server-status  - Estado de servicios en servidor"
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

# Ejecutar con logs completos (Loki + Grafana)
docker-run-logs:
	@echo "🚀 Iniciando stack completo con logs..."
	docker-compose -f docker-compose.logging.yml up --build -d
	@echo "✅ Stack con logs iniciado"
	@echo "📊 Grafana: http://localhost:3000 (admin/admin)"
	@echo "📋 Loki: http://localhost:3100"
	@echo "🔍 Promtail recopilando logs automáticamente"

# Detener stack con logs
docker-stop-logs:
	@echo "🛑 Deteniendo stack con logs..."
	docker-compose -f docker-compose.logging.yml down --volumes
	@echo "✅ Stack con logs detenido"

# Ver logs de contenedores específicos
logs-proxy:
	docker-compose logs -f proxy1 proxy2 proxy3 proxy4

# Ver logs de Loki
logs-loki:
	docker-compose -f docker-compose.logging.yml logs -f loki

# Ver logs de Promtail  
logs-promtail:
	docker-compose -f docker-compose.logging.yml logs -f promtail

# Abrir Grafana en el navegador (macOS)
open-grafana:
	@echo "🌐 Abriendo Grafana en el navegador..."
	open http://localhost:3000

# Test directo a Loki API
test-loki:
	@echo "🔍 Verificando API de Loki..."
	curl -s http://localhost:3100/ready && echo "✅ Loki ready" || echo "❌ Loki no disponible"

# === COMANDOS DE DEPLOYMENT ===

# Despliegue completo en servidor
deploy-server:
	@echo "🚀 Desplegando en servidor de producción..."
	chmod +x deployment/deploy-all.sh
	./deployment/deploy-all.sh
	@echo "✅ Despliegue completado"

# Configurar solo el servidor (primera vez)
setup-server:
	@echo "🔧 Configurando servidor inicial..."
	scp deployment/setup-server.sh root@137.184.47.82:/tmp/
	ssh root@137.184.47.82 "chmod +x /tmp/setup-server.sh && /tmp/setup-server.sh"
	@echo "✅ Servidor configurado"

# Configurar solo Jenkins
setup-jenkins:
	@echo "⚙️ Configurando Jenkins..."
	scp deployment/configure-jenkins.sh root@137.184.47.82:/tmp/
	ssh root@137.184.47.82 "chmod +x /tmp/configure-jenkins.sh && /tmp/configure-jenkins.sh"
	@echo "✅ Jenkins configurado"

# Actualizar aplicación en servidor
deploy-app:
	@echo "📦 Actualizando aplicación en servidor..."
	@echo "🏗️ Building imagen..."
	docker build -t $(DOCKER_IMAGE):latest .
	docker save $(DOCKER_IMAGE):latest > meli-proxy-latest.tar
	@echo "📤 Copiando al servidor..."
	scp meli-proxy-latest.tar root@137.184.47.82:/tmp/
	scp deployment/docker-compose.prod.yml root@137.184.47.82:/opt/meli-proxy/docker-compose.logging.yml
	@echo "🚀 Desplegando..."
	ssh root@137.184.47.82 "cd /opt/meli-proxy && \
		docker load < /tmp/meli-proxy-latest.tar && \
		docker-compose -f docker-compose.logging.yml down --remove-orphans && \
		docker-compose -f docker-compose.logging.yml up -d && \
		rm /tmp/meli-proxy-latest.tar"
	rm meli-proxy-latest.tar
	@echo "✅ Aplicación actualizada"

# Health check del servidor
check-server:
	@echo "🏥 Verificando salud del servidor..."
	@curl -f http://137.184.47.82/health && echo "✅ Aplicación OK" || echo "❌ Aplicación con problemas"
	@curl -f http://137.184.47.82/metrics | grep -q meli_proxy && echo "✅ Métricas OK" || echo "❌ Métricas con problemas"
	@echo "🌐 Servicios disponibles:"
	@echo "   • App: http://137.184.47.82"
	@echo "   • Jenkins: http://137.184.47.82/jenkins/"
	@echo "   • Grafana: http://137.184.47.82/grafana/"

# Logs del servidor
server-logs:
	@echo "📋 Obteniendo logs del servidor..."
	ssh root@137.184.47.82 "cd /opt/meli-proxy && docker-compose -f docker-compose.logging.yml logs --tail=100"

# Reiniciar servicios en servidor
restart-server:
	@echo "🔄 Reiniciando servicios en servidor..."
	ssh root@137.184.47.82 "cd /opt/meli-proxy && docker-compose -f docker-compose.logging.yml restart"
	@echo "✅ Servicios reiniciados"

# Status del servidor
server-status:
	@echo "📊 Estado del servidor..."
	ssh root@137.184.47.82 "cd /opt/meli-proxy && docker-compose -f docker-compose.logging.yml ps"
