#!/bin/bash

echo "🚀 INICIANDO MELI-PROXY OPTIMIZADO"
echo "=================================="

echo "Compilando imagen optimizada..."
docker build -t meli-proxy-optimized .

echo "Iniciando servicios..."
docker compose up -d

echo ""
echo "⏳ Esperando que los servicios estén listos..."
sleep 5

echo ""
echo "✅ SERVICIOS LISTOS!"
echo "==================="
echo "Load Balancer: http://localhost:8080"
echo "Métricas: http://localhost:8081/metrics"
echo "Grafana: http://localhost:3000"

echo ""
echo "🧪 Prueba básica:"
curl -s -o /dev/null -w "Status: %{http_code}\n" http://localhost:8080/sites/MLA

echo ""
echo "📊 Ver logs:"
echo "docker compose logs -f"
echo ""
echo "🛑 Detener:"
echo "docker compose down"
