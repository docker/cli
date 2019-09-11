wrappedNode(label: 'linux && x86_64', cleanWorkspace: true) {
  timeout(time: 60, unit: 'MINUTES') {
    stage "Git Checkout"
    checkout scm

    stage "Run end-to-end test suite"
    sh "docker version"
    sh "docker info"
    sh "E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
        IMAGE_TAG=clie2e${BUILD_NUMBER} \
        DOCKER_BUILDKIT=1 make -f docker.Makefile test-e2e"
  }
}
