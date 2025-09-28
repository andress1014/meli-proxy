#!/bin/bash
set -e

echo "🚀 Configurando servidor Ubuntu para meli-proxy CI/CD"

# Actualizar sistema
echo "📦 Actualizando sistema..."
apt update && apt upgrade -y

# Instalar dependencias básicas
echo "🔧 Instalando dependencias básicas..."
apt install -y \
    curl \
    wget \
    git \
    unzip \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release \
    jq \
    htop \
    ufw

# Configurar firewall básico
echo "🔒 Configurando firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 8080/tcp
ufw allow 8081/tcp
ufw allow 3000/tcp
ufw allow 8082/tcp  # Jenkins
ufw --force enable

# Instalar Docker
echo "🐳 Instalando Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Configurar Docker
systemctl enable docker
systemctl start docker
usermod -aG docker root

# Instalar Docker Compose standalone
echo "📦 Instalando Docker Compose..."
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose

# Instalar Jenkins
echo "⚙️  Instalando Jenkins..."
curl -fsSL https://pkg.jenkins.io/debian-stable/jenkins.io-2023.key | sudo tee /usr/share/keyrings/jenkins-keyring.asc > /dev/null
echo deb [signed-by=/usr/share/keyrings/jenkins-keyring.asc] https://pkg.jenkins.io/debian-stable binary/ | sudo tee /etc/apt/sources.list.d/jenkins.list > /dev/null
apt update
apt install -y openjdk-17-jdk jenkins

# Configurar Jenkins
systemctl enable jenkins
systemctl start jenkins

# Agregar jenkins al grupo docker
usermod -aG docker jenkins
systemctl restart jenkins

# Instalar Nginx
echo "🌐 Instalando Nginx..."
apt install -y nginx
systemctl enable nginx
systemctl start nginx

# Crear directorio para el proyecto
echo "📁 Creando directorios..."
mkdir -p /opt/meli-proxy
mkdir -p /opt/meli-proxy/logs
mkdir -p /var/log/meli-proxy
chown -R jenkins:jenkins /opt/meli-proxy
chmod -R 755 /opt/meli-proxy

# Configurar Git para Jenkins
echo "🔧 Configurando Git..."
git config --global user.name "Jenkins CI"
git config --global user.email "jenkins@meli-proxy.com"

echo "✅ Configuración del servidor completada!"
echo ""
echo "📋 Información importante:"
echo "🔑 Password inicial de Jenkins: $(cat /var/lib/jenkins/secrets/initialAdminPassword)"
echo "🌐 Jenkins URL: http://137.184.47.82:8080"
echo "🐳 Docker version: $(docker --version)"
echo "⚙️  Jenkins status: $(systemctl is-active jenkins)"
echo ""
echo "🚀 Siguiente paso: Configurar Jenkins en http://137.184.47.82:8080"
