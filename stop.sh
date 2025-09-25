#!/bin/bash

echo "🛑 DETENIENDO MELI-PROXY"
echo "========================"

echo "Deteniendo servicios..."
docker compose down

echo ""
echo "✅ Servicios detenidos correctamente!"
echo ""
echo "🔄 Para reiniciar:"
echo "./start.sh"
