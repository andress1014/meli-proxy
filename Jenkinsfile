pipeline {
  agent any
  stages {
    stage('Checkout') {
      steps {
        sh 'git --version'
        git url: 'https://github.com/andress1014/meli-proxy.git', branch: 'main'
      }
    }
    stage('Build') {
      steps {
        sh 'ls -la && echo OK'
      }
    }
  }
}