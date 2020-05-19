pipeline {
    agent {
        label "linux && x86_64"
    }

    options {
        timeout(time: 60, unit: 'MINUTES')
    }

    stages {
        stage("Docker info") {
            steps {
                sh "docker version"
                sh "docker info"
            }
        }
        stage("E2E Tests - stable engine") {
            steps {
                sh "E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
                  IMAGE_TAG=clie2e${BUILD_NUMBER} \
                  DOCKER_BUILDKIT=1 make -f docker.Makefile test-e2e"
            }
        }
        stage("E2E Tests - 18.09 engine") {
            steps {
                sh "E2E_ENGINE_VERSION=18.09-dind \
                  E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
                  IMAGE_TAG=clie2e${BUILD_NUMBER} \
                  DOCKER_BUILDKIT=1 make -f docker.Makefile test-e2e"
            }
        }
    }
}
