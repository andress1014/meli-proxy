#!/bin/bash
set -e

echo "⚙️ Configurando Jenkins para meli-proxy"

# Configurar Nginx para Jenkins
echo "🌐 Configurando Nginx..."
cat > /etc/nginx/sites-available/meli-proxy << 'EOF'
server {
    listen 80;
    server_name 137.184.47.82;

    # Proxy para meli-proxy
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts optimizados
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Jenkins (puerto alternativo)
    location /jenkins/ {
        proxy_pass http://localhost:8082/jenkins/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Headers específicos para Jenkins
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_redirect http://localhost:8082/jenkins/ /jenkins/;
    }

    # Métricas
    location /metrics {
        proxy_pass http://localhost:9090/metrics;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # Health check
    location /health {
        proxy_pass http://localhost:8080/health;
        proxy_set_header Host $host;
        access_log off;
    }

    # Grafana
    location /grafana/ {
        proxy_pass http://localhost:3000/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF

# Habilitar sitio
ln -sf /etc/nginx/sites-available/meli-proxy /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

# Configurar Jenkins para usar puerto 8082
echo "🔧 Configurando Jenkins..."
sed -i 's/HTTP_PORT=8080/HTTP_PORT=8082/g' /etc/default/jenkins
sed -i 's#JENKINS_ARGS="--webroot.*#JENKINS_ARGS="--webroot=/var/cache/$NAME/war --httpPort=$HTTP_PORT --prefix=/jenkins"#g' /etc/default/jenkins

systemctl restart jenkins

# Esperar a que Jenkins esté listo
echo "⏳ Esperando que Jenkins esté listo..."
sleep 30

# Instalar plugins básicos de Jenkins
echo "🔌 Configurando plugins de Jenkins..."

# Obtener password inicial
JENKINS_PASSWORD=$(cat /var/lib/jenkins/secrets/initialAdminPassword)

# Función para hacer requests a Jenkins
jenkins_cli() {
    java -jar /var/cache/jenkins/war/WEB-INF/jenkins-cli.jar -s http://localhost:8082/jenkins/ -auth admin:$JENKINS_PASSWORD "$@"
}

# Descargar Jenkins CLI
wget -q http://localhost:8082/jenkins/jnlpJars/jenkins-cli.jar -O /var/cache/jenkins/war/WEB-INF/jenkins-cli.jar

# Instalar plugins esenciales
echo "📦 Instalando plugins..."
jenkins_cli install-plugin git
jenkins_cli install-plugin workflow-aggregator
jenkins_cli install-plugin docker-workflow
jenkins_cli install-plugin pipeline-stage-view
jenkins_cli install-plugin blueocean
jenkins_cli install-plugin github
jenkins_cli install-plugin ws-cleanup

# Reiniciar Jenkins para aplicar plugins
jenkins_cli restart

echo "⏳ Esperando reinicio de Jenkins..."
sleep 60

# Crear job de pipeline
echo "🚀 Creando job de pipeline..."
cat > /tmp/meli-proxy-pipeline.xml << 'EOF'
<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.40">
  <actions/>
  <description>Pipeline de CI/CD para meli-proxy</description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty>
      <triggers>
        <com.cloudbees.jenkins.GitHubPushTrigger plugin="github@1.34.1">
          <spec></spec>
        </com.cloudbees.jenkins.GitHubPushTrigger>
      </triggers>
    </org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsScmFlowDefinition" plugin="workflow-cps@2.92">
    <scm class="hudson.plugins.git.GitSCM" plugin="git@4.8.3">
      <configVersion>2</configVersion>
      <userRemoteConfigs>
        <hudson.plugins.git.UserRemoteConfig>
          <url>https://github.com/andress1014/meli-proxy.git</url>
        </hudson.plugins.git.UserRemoteConfig>
      </userRemoteConfigs>
      <branches>
        <hudson.plugins.git.BranchSpec>
          <name>*/main</name>
        </hudson.plugins.git.BranchSpec>
      </branches>
      <doGenerateSubmoduleConfigurations>false</doGenerateSubmoduleConfigurations>
      <submoduleCfg class="list"/>
      <extensions/>
    </scm>
    <scriptPath>Jenkinsfile</scriptPath>
    <lightweight>true</lightweight>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>
EOF

# Crear el job
jenkins_cli create-job meli-proxy-pipeline < /tmp/meli-proxy-pipeline.xml
rm /tmp/meli-proxy-pipeline.xml

# Configurar webhook (manual)
echo ""
echo "✅ Jenkins configurado exitosamente!"
echo ""
echo "📋 Información de acceso:"
echo "🌐 Jenkins URL: http://137.184.47.82/jenkins/"
echo "👤 Usuario: admin"
echo "🔑 Password: $JENKINS_PASSWORD"
echo ""
echo "🔧 Pasos manuales restantes:"
echo "1. Ir a http://137.184.47.82/jenkins/"
echo "2. Completar configuración inicial"
echo "3. Configurar webhook en GitHub:"
echo "   - URL: http://137.184.47.82/jenkins/github-webhook/"
echo "   - Content type: application/json"
echo "   - Events: Push events"
echo ""
echo "🚀 Pipeline 'meli-proxy-pipeline' creado y listo!"
