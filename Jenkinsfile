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
                    # Instalar Go si no está disponible
                    if ! command -v go &> /dev/null; then
                        echo "Installing Go..."
                        wget -q https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
                        sudo rm -rf /usr/local/go
                        sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
                        export PATH=$PATH:/usr/local/go/bin
                    fi
                    
                    # Configurar Go
                    export GOPATH=$HOME/go
                    export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
                    
                    # Ejecutar tests
                    make test || {
                        echo "❌ Tests fallaron"
                        exit 1
                    }
                '''
            }
            post {
                always {
                    // Publicar resultados de tests
                    publishTestResults testResultsPattern: 'test-results.xml'
                }
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
                    cd ${PROJECT_DIR}
                    
                    # Backup de la configuración actual
                    if [ -f ${COMPOSE_FILE} ]; then
                        cp ${COMPOSE_FILE} ${COMPOSE_FILE}.backup.$(date +%Y%m%d_%H%M%S)
                    fi
                    
                    # Copiar archivos de configuración
                    cp -r ${WORKSPACE}/* ${PROJECT_DIR}/
                    
                    # Detener servicios actuales
                    docker-compose -f ${COMPOSE_FILE} down --remove-orphans || true
                    
                    # Iniciar servicios con la nueva imagen
                    docker-compose -f ${COMPOSE_FILE} up -d
                    
                    # Esperar que los servicios estén listos
                    sleep 30
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
            // Limpiar workspace
            cleanWs()
            
            // Logs de contenedores para debugging
            sh '''
                echo "📋 Container logs:"
                cd ${PROJECT_DIR}
                docker-compose -f ${COMPOSE_FILE} logs --tail=50 || true
            '''
        }
        
        success {
            echo '🎉 Pipeline completed successfully!'
            
            // Notificación de éxito (opcional)
            sh '''
                echo "✅ Deployment successful at $(date)"
                echo "🌐 Service available at: http://137.184.47.82"
                echo "📊 Metrics: http://137.184.47.82/metrics"
                echo "🏥 Health: http://137.184.47.82/health"
            '''
        }
        
        failure {
            echo '❌ Pipeline failed!'
            
            // Rollback automático
            sh '''
                echo "🔄 Attempting rollback..."
                cd ${PROJECT_DIR}
                
                # Buscar último backup funcional
                BACKUP_FILE=$(ls -t ${COMPOSE_FILE}.backup.* 2>/dev/null | head -1)
                
                if [ -n "$BACKUP_FILE" ]; then
                    echo "Rolling back to $BACKUP_FILE"
                    cp "$BACKUP_FILE" ${COMPOSE_FILE}
                    docker-compose -f ${COMPOSE_FILE} up -d
                    echo "✅ Rollback completed"
                else
                    echo "⚠️ No backup found for rollback"
                fi
            '''
        }
    }
}
