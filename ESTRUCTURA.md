# 🗂️ ESTRUCTURA FINAL DEL PROYECTO

## 📁 Archivos Principales

### 🐳 Docker & Configuración
- `docker-compose.yml` - **Configuración optimizada para 50K RPS** (4 instancias + load balancer)
- `docker-compose.basic.yml` - Configuración básica para desarrollo (1 instancia)
- `Dockerfile` - Imagen optimizada multi-stage
- `nginx.conf` - Load balancer optimizado para alta concurrencia
- `prometheus.yml` - Configuración de métricas

### 🚀 Scripts de Administración  
- `start.sh` - **Script principal** para iniciar todo el sistema
- `setup.sh` - Configuración completa para pruebas de 50K RPS
- `monitor.sh` - Monitoreo en tiempo real del sistema

### 🧪 Scripts de Testing
- `test-basic.sh` - Pruebas básicas de funcionalidad y latencia
- `test-performance.sh` - Pruebas extremas de performance (hasta 50K RPS)

### 📚 Documentación
- `README.md` - Documentación principal actualizada
- `PERFORMANCE-OPTIMIZATIONS.md` - Detalles de optimizaciones

## 🎯 Uso Recomendado

### Para Desarrollo:
```bash
docker compose -f docker-compose.basic.yml up
```

### Para Producción/Testing:
```bash
./start.sh
# o
docker compose up -d
```

### Para Pruebas de Performance:
```bash
./test-basic.sh      # Pruebas básicas
./test-performance.sh # Pruebas extremas
./monitor.sh         # Monitoreo en tiempo real
```

## 📊 Endpoints Disponibles

- **Proxy**: http://localhost:8080
- **Métricas**: http://localhost:8081/metrics  
- **Prometheus**: http://localhost:9091
- **Load Balancer Status**: http://localhost:8081/nginx_status

¡Sistema listo para 50K RPS! 🚀
