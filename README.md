# meli-proxy

Un proxy transparente optimizado para API de MercadoLibre con rate limiting distribuido, métricas avanzadas y capacidad para **50K RPS**.

## 🚀 Características

- **Proxy Transparente**: Reenvía todas las requests a `api.mercadolibre.com` sin modificar headers o redirects
- **Rate Limiting Distribuido**: Límites por IP, path y combinación IP+path usando sliding window en Redis
- **Arquitectura Escalable**: 4 instancias con load balancer Nginx optimizado para alta concurrencia
- **Métricas Prometheus**: Monitoreo completo de requests, latencias y rate limits
- **Optimizado para 50K RPS**: Configuración de alto rendimiento con connection pooling masivo
- **Extracción de IP Robusta**: Maneja `X-Forwarded-For`, `X-Real-IP` y `RemoteAddr`
- **Alta Performance**: Pool de conexiones HTTP/2, keep-alive y timeouts optimizados
- **Apagado Elegante**: Graceful shutdown con timeout configurable

## 🏗️ Arquitectura

```
┌─────────────┐   ┌──────────────┐   ┌─────────────┐   ┌──────────────────┐
│   Client    │──▶│  meli-proxy  │──▶│   Redis     │   │ api.mercadolibre │
│             │   │              │   │ (rate limit)│   │      .com        │
└─────────────┘   └──────────────┘   └─────────────┘   └──────────────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │ Prometheus  │
                  │ (métricas)  │
                  └─────────────┘
```

## 🛠️ Instalación y Uso

### Inicio Rápido

```bash
# Clonar el repositorio
git clone https://github.com/andress1014/meli-proxy.git
cd meli-proxy

# Iniciar configuración optimizada (4 instancias + load balancer)
./start.sh

# O manualmente
docker compose up --build -d
```

### Configuraciones Disponibles

- **`docker-compose.yml`**: Configuración optimizada para 50K RPS (4 instancias)
- **`docker-compose.basic.yml`**: Configuración básica para desarrollo (1 instancia)

### Desarrollo Local

```bash
# Instalar dependencias
make deps

# Iniciar Redis
make redis

# Ejecutar la aplicación
make run
```

## 🔧 Configuración

### Variables de Entorno

| Variable | Descripción | Valor por Defecto |
|----------|-------------|-------------------|
| `PORT` | Puerto del servidor proxy | `8080` |
| `METRICS_PORT` | Puerto del servidor de métricas | `9090` |
| `TARGET_URL` | URL de destino | `https://api.mercadolibre.com` |
| `REDIS_URL` | URL de conexión a Redis | `redis://localhost:6379` |
| `LOG_LEVEL` | Nivel de logging | `info` |
| `DEFAULT_RPS` | Rate limit por defecto (req/min) | `100` |
| `IP_RATE_LIMITS` | Límites por IP específica | `""` |
| `PATH_RATE_LIMITS` | Límites por path específico | `""` |
| `IP_PATH_RATE_LIMITS` | Límites por IP+path específico | `""` |

### Ejemplos de Rate Limits

```bash
# Límites por IP
IP_RATE_LIMITS="192.168.1.100:200,10.0.0.1:50"

# Límites por path
PATH_RATE_LIMITS="/categories/*:500,/items/*:300,/users/*:100"

# Límites por IP+path
IP_PATH_RATE_LIMITS="192.168.1.100::/categories/*:100,10.0.0.1::/items/*:50"
```

## 📊 Rate Limiting

### Algoritmo Sliding Window

- Usa Redis con script Lua atómico
- Ventana deslizante de 60 segundos
- Tres niveles de limitación:
  - **IP**: `ip::<A.B.C.D>`
  - **Path**: `path::<pattern>`
  - **IP+Path**: `ip_path::<A.B.C.D>::<pattern>`

### Normalización de Paths

| Path Original | Path Normalizado |
|---------------|------------------|
| `/categories/MLA1234` | `/categories/*` |
| `/items/MLA123456789` | `/items/*` |
| `/users/123456` | `/users/*` |
| `/sites/MLA` | `/sites/*` |

## 📈 Métricas

### Endpoints

- `http://localhost:9090/metrics` - Métricas Prometheus
- `http://localhost:8080/health` - Health check

### Métricas Disponibles

- `meli_proxy_requests_total` - Total de requests por método, path y status
- `meli_proxy_rate_limit_blocked_total` - Requests bloqueados por rate limit
- `meli_proxy_request_duration_seconds` - Latencias de requests
- `meli_proxy_requests_in_progress` - Requests en progreso
- `meli_proxy_requests_per_second` - RPS actual por path

## 🧪 Testing

```bash
# Ejecutar tests
make test

# Load test básico
make load-test

# Ver métricas
make metrics

# Check health
make health
```

## 🐳 Docker

```bash
# Build imagen
make docker-build

# Ejecutar con compose
make docker-run

# Ver logs
make logs

# Parar servicios
make stop
```

## 📝 Comandos Útiles

```bash
# Test básico con categoría válida
curl http://localhost:8080/categories/MLA120352

# Test con headers de rate limit
curl -I http://localhost:8080/items/MLA123456

# Ver métricas
curl http://localhost:9090/metrics | grep meli_proxy

# Health check
curl http://localhost:8080/health
```

## 🧪 Testing

El proyecto incluye una suite completa de tests organizados en la carpeta `tests/`.

### Comandos de Testing

```bash
# Ejecutar todos los tests
make test

# Solo tests unitarios
make test-unit

# Solo tests de integración
make test-integration

# Tests con coverage report
make test-coverage

# Tests con race detection
make test-race

# Benchmarks de performance
make test-bench
```

### Estructura de Tests

```
tests/
├── unit/                   # Tests unitarios
│   ├── config_test.go     # Tests de configuración
│   ├── httpclient_test.go # Tests del cliente HTTP
│   ├── metrics_test.go    # Tests de métricas
│   └── ratelimit_utils_test.go # Tests de rate limiting
└── integration/            # Tests de integración
    └── proxy_integration_test.go # Tests end-to-end
```

### Cobertura de Tests

- ✅ **Configuración**: Parsing de variables de entorno y validación
- ✅ **HTTP Client**: Optimizaciones de conexiones y manejo de redirects  
- ✅ **Métricas**: Prometheus metrics y async recording
- ✅ **Rate Limiting**: Extracción de IP y normalización de paths
- ✅ **Integración**: Tests end-to-end con servidor real

## 🏆 Rendimiento

- **Throughput**: 10,000+ RPS en hardware moderno
- **Latency**: <5ms P99 para requests locales
- **Memory**: ~50MB en uso normal
- **Connections**: Pool reutilizable con HTTP/2

## 🛡️ Características de Producción

- Graceful shutdown con timeout
- Health checks integrados
- Métricas detalladas para monitoreo
- Logs estructurados JSON
- Rate limiting distribuido con Redis
- Circuit breaker implícito en errores upstream
- Headers informativos de rate limit
- Fail-open en errores de Redis

## 📂 Estructura del Proyecto

```
├── cmd/proxy/           # Aplicación principal
├── internal/
│   ├── config/         # Configuración
│   ├── logger/         # Logging con zap
│   ├── metrics/        # Métricas Prometheus
│   ├── middleware/     # Rate limiting y métricas
│   ├── proxy/          # Servidor proxy
│   └── ratelimit/      # Redis sliding window
├── pkg/httpclient/     # Cliente HTTP optimizado
├── docker-compose.yml  # Entorno de desarrollo
├── Dockerfile         # Imagen de producción
├── Makefile          # Comandos útiles
└── prometheus.yml    # Configuración Prometheus
```

## 🤝 Contribución

1. Fork el proyecto
2. Crear branch de feature (`git checkout -b feature/amazing-feature`)
3. Commit cambios (`git commit -m 'Add amazing feature'`)
4. Push al branch (`git push origin feature/amazing-feature`)
5. Abrir Pull Request

## 📄 Licencia

Este proyecto está bajo la licencia MIT - ver el archivo [LICENSE](LICENSE) para detalles.
