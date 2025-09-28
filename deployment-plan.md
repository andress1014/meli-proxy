# Plan de Deployment - MeLi Proxy

## ✅ Desarrollo Local Completado

### Mejoras Implementadas:
1. **Seguridad mejorada:**
   - Middleware de seguridad con headers HTTP
   - Validación de requests
   - Soporte para CORS
   - Health endpoint con control de acceso por IP

2. **Rate Limiting flexible:**
   - Soporte para DummyLimiter (sin limiting)
   - Interfaz común para diferentes tipos de limiters
   - Configuración via variables de entorno

3. **Docker optimizado:**
   - Imagen distroless para seguridad
   - Compilación estática
   - Variables de entorno configurables

### Resultados de Testing Local:
- ✅ **3,670 RPS** exitosos
- ✅ **13.6ms tiempo promedio** 
- ✅ **1000 requests procesados** en 0.27 segundos
- ✅ **Contenedor funcionando correctamente**

## 🚀 Plan de Deployment al Servidor Remoto

### Pasos a seguir:

1. **Subir imagen Docker al servidor remoto**
   ```bash
   # Opción 1: Registry público/privado
   docker tag meli-proxy:latest your-registry/meli-proxy:latest
   docker push your-registry/meli-proxy:latest
   
   # Opción 2: Construcción directa en servidor
   scp -r . root@137.184.47.82:/tmp/meli-proxy/
   ssh root@137.184.47.82 "cd /tmp/meli-proxy && docker build -t meli-proxy:latest ."
   ```

2. **Detener el servicio actual**
   ```bash
   ssh root@137.184.47.82 "systemctl stop meli-proxy"
   ```

3. **Ejecutar el nuevo contenedor**
   ```bash
   ssh root@137.184.47.82 "docker run -d --restart=unless-stopped -p 8080:8080 --name meli-proxy-prod -e REDIS_ENABLED=false -e DEFAULT_RPS=10000 meli-proxy:latest"
   ```

4. **Verificar funcionamiento**
   ```bash
   curl http://137.184.47.82:8080/health
   ```

5. **Testing con alta carga**
   ```bash
   # Test progresivo
   ab -n 100 -c 10 http://137.184.47.82:8080/health
   ab -n 1000 -c 100 http://137.184.47.82:8080/health
   ```

### Variables de entorno recomendadas para producción:
- `REDIS_ENABLED=false` (usar DummyLimiter para máximo rendimiento)
- `DEFAULT_RPS=10000` (límite alto)
- `LOG_LEVEL=warn` (menos logs para mejor performance)
- `GOMAXPROCS=4` (ajustar según CPU del servidor)

### Configuración nginx (opcional):
Si queremos mantener nginx como proxy reverso:
```nginx
upstream meli-proxy {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    location / {
        proxy_pass http://meli-proxy;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

## 📊 Expectativas de Performance

Basado en los tests locales, esperamos:
- **> 2,000 RPS** en el servidor remoto
- **< 50ms tiempo de respuesta**
- **0% error rate** con DummyLimiter
- **Utilización eficiente de recursos**

¿Procedemos con el deployment?
