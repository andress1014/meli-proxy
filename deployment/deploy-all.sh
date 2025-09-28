#!/bin/bash
set -e

echo "🚀 Ejecutando instalación completa en el servidor"

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Variables
SERVER_HOST="137.184.47.82"
SERVER_USER="root"
SSH_KEY="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHMcaSum8leXHPh4o9hqIosJVgykIM1as2jdgnrtUc2d deploy-meli"

print_status "Configurando servidor $SERVER_HOST..."

# Verificar conectividad
print_status "Verificando conectividad SSH..."
if ! ssh -o ConnectTimeout=10 -o BatchMode=yes "$SERVER_USER@$SERVER_HOST" echo "SSH OK"; then
    print_error "No se puede conectar al servidor via SSH"
    print_warning "Verifica que la clave SSH esté configurada correctamente"
    exit 1
fi

print_success "Conectividad SSH verificada"

# Copiar scripts al servidor
print_status "Copiando archivos de configuración al servidor..."
scp -r deployment/ "$SERVER_USER@$SERVER_HOST":/tmp/
scp docker-compose.logging.yml "$SERVER_USER@$SERVER_HOST":/tmp/deployment/
scp nginx.conf "$SERVER_USER@$SERVER_HOST":/tmp/deployment/
scp loki-config.yml "$SERVER_USER@$SERVER_HOST":/tmp/deployment/
scp promtail-config.yml "$SERVER_USER@$SERVER_HOST":/tmp/deployment/

# Ejecutar configuración en el servidor
print_status "Ejecutando configuración del servidor..."
ssh "$SERVER_USER@$SERVER_HOST" << 'EOF'
    set -e
    
    cd /tmp/deployment
    chmod +x *.sh
    
    echo "🔧 Ejecutando setup del servidor..."
    ./setup-server.sh
    
    echo "⚙️ Configurando Jenkins..."
    ./configure-jenkins.sh
    
    echo "📁 Configurando directorio del proyecto..."
    cp docker-compose.prod.yml /opt/meli-proxy/docker-compose.logging.yml
    cp nginx.conf /opt/meli-proxy/
    cp loki-config.yml /opt/meli-proxy/
    cp promtail-config.yml /opt/meli-proxy/
    cp nginx-meli-proxy.conf /etc/nginx/sites-available/meli-proxy
    
    echo "🌐 Configurando Nginx..."
    ln -sf /etc/nginx/sites-available/meli-proxy /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
    nginx -t && systemctl reload nginx
    
    echo "🧹 Limpiando archivos temporales..."
    rm -rf /tmp/deployment
    
    echo "✅ Configuración completada!"
EOF

print_success "Servidor configurado exitosamente!"

# Obtener información de Jenkins
JENKINS_PASSWORD=$(ssh "$SERVER_USER@$SERVER_HOST" "cat /var/lib/jenkins/secrets/initialAdminPassword 2>/dev/null || echo 'No disponible'")

print_status "Configuración completada. Información de acceso:"
echo ""
echo "🌐 Servicios disponibles:"
echo "   • Aplicación: http://$SERVER_HOST"
echo "   • Jenkins: http://$SERVER_HOST/jenkins/"
echo "   • Grafana: http://$SERVER_HOST/grafana/"
echo "   • Métricas: http://$SERVER_HOST/metrics"
echo "   • Health: http://$SERVER_HOST/health"
echo ""
echo "🔑 Credenciales Jenkins:"
echo "   • Usuario: admin"
echo "   • Password: $JENKINS_PASSWORD"
echo ""
echo "🔑 Credenciales Grafana:"
echo "   • Usuario: admin"
echo "   • Password: meli-proxy-admin-2024"
echo ""
print_warning "Pasos siguientes:"
echo "1. Configura el webhook en GitHub:"
echo "   URL: http://$SERVER_HOST/jenkins/github-webhook/"
echo "2. Agrega el SSH_PRIVATE_KEY a los secrets de GitHub"
echo "3. Haz push al repositorio para activar el pipeline"
echo ""
print_success "🎉 ¡Instalación completada!"
