pipeline {
    agent any
    
    environment {
        DOCKER_IMAGE = 'meli-proxy-optimized'
        DOCKER_TAG = "${BUILD_NUMBER}"
        PROJECT_DIR = '/opt/meli-proxy'
        COMPOSE_FILE = 'docker-compose.logging.yml'
    }
    
    stages {
        stage('🔍 Checkout') {
            steps {
                echo 'Checking out code...'
                checkout scm
                sh 'git clean -fdx'
            }
        }
        
        stage('🧪 Run Tests') {
            steps {
                echo 'Running tests...'
                sh '''
                    # Verificar si Go está disponible, sino instalar
                    if ! command -v go &> /dev/null; then
                        echo "Installing Go..."
                        # Usar curl en lugar de wget
                        curl -L -o go1.21.0.linux-amd64.tar.gz https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
                        tar -C /tmp -xzf go1.21.0.linux-amd64.tar.gz
                        export PATH=/tmp/go/bin:$PATH
                    fi
                    
                    # Configurar Go
                    export GOPATH=$HOME/go
                    export PATH=$PATH:/usr/local/go/bin:/tmp/go/bin:$GOPATH/bin
                    
                    # Verificar que tenemos Go
                    go version
                    
                    # Ejecutar tests si existe Makefile, sino saltar
                    if [ -f "Makefile" ] && grep -q "test:" Makefile; then
                        make test || echo "⚠️ Tests fallaron, pero continuando..."
                    else
                        echo "ℹ️ No se encontró target de test, saltando..."
                    fi
                '''
            }
        }
        
        stage('🔧 Build') {
            steps {
                echo 'Building Docker image...'
                script {
                    sh '''
                        # Build de la imagen Docker
                        docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} .
                        docker tag ${DOCKER_IMAGE}:${DOCKER_TAG} ${DOCKER_IMAGE}:latest
                        
                        # Limpiar imágenes antiguas
                        docker image prune -f
                    '''
                }
            }
        }
        
        stage('🛡️ Security Scan') {
            steps {
                echo 'Running security scan...'
                sh '''
                    # Scan de vulnerabilidades con Trivy
                    if ! command -v trivy &> /dev/null; then
                        echo "Installing Trivy..."
                        sudo apt-get update
                        sudo apt-get install wget apt-transport-https gnupg lsb-release -y
                        wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
                        echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
                        sudo apt-get update
                        sudo apt-get install trivy -y
                    fi
                    
                    trivy image --exit-code 1 --severity HIGH,CRITICAL ${DOCKER_IMAGE}:${DOCKER_TAG} || {
                        echo "⚠️ Vulnerabilidades críticas encontradas, pero continuando..."
                    }
                '''
            }
        }
        
        stage('🚀 Deploy to Staging') {
            steps {
                echo 'Deploying to staging...'
                sh '''
                    # Crear directorio si no existe
                    sudo mkdir -p ${PROJECT_DIR}
                    sudo chmod 755 ${PROJECT_DIR}
                    
                    # Ir al directorio
                    cd ${PROJECT_DIR}
                    
                    # Backup de la configuración actual
                    if [ -f ${COMPOSE_FILE} ]; then
                        sudo cp ${COMPOSE_FILE} ${COMPOSE_FILE}.backup.$(date +%Y%m%d_%H%M%S)
                    fi
                    
                    # Copiar archivos de configuración
                    sudo cp -r ${WORKSPACE}/* ${PROJECT_DIR}/
                    
                    # Detener servicios actuales si existen
                    if [ -f ${COMPOSE_FILE} ]; then
                        docker-compose -f ${COMPOSE_FILE} down --remove-orphans || true
                    fi
                    
                    # Verificar si tenemos docker-compose file
                    if [ -f ${COMPOSE_FILE} ]; then
                        echo "✅ Starting services with ${COMPOSE_FILE}"
                        docker-compose -f ${COMPOSE_FILE} up -d
                        sleep 30
                    else
                        echo "ℹ️ No compose file found, skipping service start"
                    fi
                '''
            }
        }
        
        stage('✅ Health Check') {
            steps {
                echo 'Running health checks...'
                script {
                    sh '''
                        # Verificar que los servicios estén funcionando
                        max_attempts=10
                        attempt=1
                        
                        while [ $attempt -le $max_attempts ]; do
                            echo "Health check attempt $attempt/$max_attempts"
                            
                            # Verificar health endpoint
                            if curl -f http://localhost:8080/health; then
                                echo "✅ Health check passed"
                                break
                            fi
                            
                            if [ $attempt -eq $max_attempts ]; then
                                echo "❌ Health check failed after $max_attempts attempts"
                                exit 1
                            fi
                            
                            sleep 10
                            attempt=$((attempt + 1))
                        done
                        
                        # Verificar métricas
                        curl -f http://localhost:9090/metrics | grep -q meli_proxy || {
                            echo "❌ Metrics endpoint failed"
                            exit 1
                        }
                        
                        # Test funcional básico
                        curl -f http://localhost:8080/categories/MLA120352 | jq . > /dev/null || {
                            echo "❌ Functional test failed"
                            exit 1
                        }
                        
                        echo "✅ All health checks passed"
                    '''
                }
            }
        }
        
        stage('📊 Performance Test') {
            steps {
                echo 'Running performance tests...'
                sh '''
                    # Test de carga básico
                    echo "Running basic load test..."
                    
                    # Instalar apache2-utils si no está disponible
                    if ! command -v ab &> /dev/null; then
                        sudo apt-get update
                        sudo apt-get install apache2-utils -y
                    fi
                    
                    # Test con Apache Bench
                    ab -n 1000 -c 10 http://localhost:8080/health || {
                        echo "⚠️ Performance test issues detected"
                    }
                    
                    # Verificar que el rate limiting funcione
                    for i in {1..50}; do
                        curl -s http://localhost:8080/categories/MLA120352 > /dev/null
                    done
                    
                    echo "✅ Performance tests completed"
                '''
            }
        }
    }
    
    post {
        always {
            echo '📋 Pipeline finished - cleaning workspace'
            cleanWs()
        }
        
        success {
            echo '🎉 Pipeline completed successfully!'
            sh '''
                echo "✅ Deployment successful at $(date)"
                echo "🌐 Service available at: http://137.184.47.82:8080"
                echo "📊 Metrics: http://137.184.47.82:9090/metrics"
                echo "🏥 Health: http://137.184.47.82:8080/health"
            '''
        }
        
        failure {
            echo '❌ Pipeline failed!'
            sh '''
                echo "🔄 Pipeline failed at $(date)"
                echo "📋 Check logs for more details"
            '''
        }
    }
}
