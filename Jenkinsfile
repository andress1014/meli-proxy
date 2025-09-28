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