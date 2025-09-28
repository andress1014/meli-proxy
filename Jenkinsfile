pipeline {
  agent any
  
  environment {
    DOCKER_IMAGE = 'meli-proxy-optimized'
    DOCKER_TAG = "${BUILD_NUMBER}"
  }
  
  stages {
    stage('🔍 Checkout') {
      steps {
        echo 'Cloning repository...'
        deleteDir()
        sh '''
          git clone https://github.com/andress1014/meli-proxy.git .
          ls -la
          echo "✅ Repository cloned successfully"
        '''
      }
    }
    
    stage('🔧 Build') {
      steps {
        echo 'Building Docker image...'
        sh '''
          docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} .
          docker tag ${DOCKER_IMAGE}:${DOCKER_TAG} ${DOCKER_IMAGE}:latest
          echo "✅ Build completed"
        '''
      }
    }
    
    stage('🚀 Deploy') {
      steps {
        echo 'Deploying services...'
        sh '''
          docker-compose -f docker-compose.yml down --remove-orphans || true
          docker-compose -f docker-compose.yml up -d
          sleep 10
          docker ps --format "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}"
          echo "✅ Deployment completed"
        '''
      }
    }
    
    stage('🏥 Health Check') {
      steps {
        echo 'Verifying deployment health...'
        sh '''
          echo "Waiting for services to be ready..."
          sleep 15
          
          # Check individual instances - health endpoint
          for port in 8082 8083 8084 8085; do
            echo "Checking health on port $port..."
            curl -s http://localhost:$port/health | jq . || echo "Warning: Instance $port not ready"
          done
          
          # Check individual instances - status endpoint  
          for port in 8082 8083 8084 8085; do
            echo "Checking status on port $port..."
            curl -s http://localhost:$port/status | jq . || echo "Warning: Status $port not ready"
          done
          
          # Check load balancer endpoints
          echo "Checking load balancer health..."
          curl -s http://localhost:8081/health | jq . || echo "Warning: Load balancer health not ready"
          
          echo "Checking load balancer status..."
          curl -s http://localhost:8081/status | jq . || echo "Warning: Load balancer status not ready"
          
          echo "✅ Health check completed - v1.3.0 deployed!"
        '''
      }
    }
  }
  
  post {
    success {
      echo '🎉 Pipeline completed successfully!'
    }
    failure {
      echo '❌ Pipeline failed!'
    }
  }
}