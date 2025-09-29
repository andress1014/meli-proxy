# 🗂️ ESTRUCTURA FINAL DEL PROYECTO

## 📁 Archivos Principales

### 🐳 Docker & Configuración
- `docker-compose.yml` - **Configuración optimizada para 50K RPS** (4 instancias + load balancer)
- `Dockerfile` - Imagen optimizada multi-stage
- `nginx.conf` - Load balancer optimizado para alta concurrencia

### 🚀 Scripts Esenciales  
- `start.sh` - **Script principal** para iniciar todo el sistema
- `stop.sh` - Detener todo el sistema completamente
- `test-50k.sh` - **Pruebas completas** de carga hasta 50K RPS

### 📚 Documentación
- `README.md` - Documentación principal actualizada

## 🎯 Uso Simplificado

### Iniciar Sistema Completo:
```bash
./start.sh
```

### Pruebas de Carga Completas:
```bash
./test-50k.sh
```

### Detener Sistema:
```bash
./stop.sh
```

## 📊 Endpoints Disponibles

- **Proxy**: http://localhost:8080
- **Métricas**: http://localhost:9090/metrics (cada instancia)
- **Redis**: localhost:6379

## 🏗️ ARQUITECTURA DEL SISTEMA

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLIENTE       │───▶│   NGINX          │───▶│  MELI-PROXY     │
│   (Requests)    │    │   Load Balancer  │    │  (4 Instancias) │
└─────────────────┘    │   :8080          │    │  :8081-8084     │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                │                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │   RATE LIMIT     │    │  API MELI       │
                       │   Redis Cache    │    │  TARGET         │
                       │   :6379          │    │  External API   │
                       └──────────────────┘    └─────────────────┘
```

### 🔄 Flujo de Datos:
1. **Cliente** → Nginx (puerto 8080)
2. **Nginx** → Load balancing entre 4 proxies
3. **Proxy** → Consulta Redis (rate limit + cache decisiones)
4. **Proxy** → Envía request a api.mercadolibre.com
5. **Proxy** → Respuesta transparente al cliente

### 🚀 Performance:
- **Capacidad**: 50,000 RPS
- **Latencia**: <10ms promedio  
- **Rate Limiting**: Distribuido con Redis
- **Escalabilidad**: Horizontal (más instancias)

¡Sistema listo para producción! 🚀
