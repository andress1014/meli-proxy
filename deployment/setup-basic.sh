#!/bin/bash
set -e

echo "🚀 Configuración básica de servidor Ubuntu para meli-proxy"

# Verificar si es root
if [[ $EUID -ne 0 ]]; then
   echo "Este script debe ejecutarse como root" 
   exit 1
fi

echo "📦 Actualizando sistema..."
apt update

echo "🔧 Instalando dependencias básicas..."
apt install -y curl wget git unzip software-properties-common \
    apt-transport-https ca-certificates gnupg lsb-release jq htop ufw

echo "🔒 Configurando firewall básico..."
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 8080/tcp
ufw allow 8081/tcp
ufw allow 3000/tcp
ufw allow 8082/tcp
ufw --force enable

echo "🐳 Instalando Docker..."
# Limpiar instalaciones previas
apt remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

# Instalar Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Configurar Docker
systemctl enable docker
systemctl start docker

# Instalar Docker Compose standalone
echo "📦 Instalando Docker Compose..."
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose

echo "🔧 Instalando Nginx..."
apt install -y nginx
systemctl enable nginx
systemctl start nginx

echo "📁 Creando directorios del proyecto..."
mkdir -p /opt/meli-proxy
mkdir -p /opt/meli-proxy/logs
mkdir -p /var/log/meli-proxy
chmod -R 755 /opt/meli-proxy

echo "✅ Configuración básica completada!"
echo ""
echo "🔍 Verificación:"
echo "Docker: $(docker --version)"
echo "Docker Compose: $(docker-compose --version)"
echo "Nginx: $(nginx -v 2>&1)"
echo ""
echo "🎯 Siguiente paso: Configurar Jenkins o desplegar la aplicación"
echo "📁 Directorio del proyecto: /opt/meli-proxy"
