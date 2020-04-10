wrappedNode(label: 'linux && x86_64', cleanWorkspace: true) {
  timeout(time: 60, unit: 'MINUTES') {
    stage "Git Checkout"
    checkout scm

    stage "Docker info"
    sh "docker version"
    sh "docker info"

    stage "e2e (non-experimental)"
    sh "E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
        IMAGE_TAG=clie2e${BUILD_NUMBER} \
        DOCKER_BUILDKIT=1 make -f docker.Makefile test-e2e-non-experimental"

    stage "e2e (experimental)"
    sh "E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
        IMAGE_TAG=clie2e${BUILD_NUMBER} \
        DOCKER_BUILDKIT=1 make -f docker.Makefile test-e2e-experimental"

    stage "e2e (ssh connhelper)"
    sh "E2E_UNIQUE_ID=clie2e${BUILD_NUMBER} \
        IMAGE_TAG=clie2e${BUILD_NUMBER} \
        DOCKER_BUILDKIT=1 make -f docker.Makefile test-e2e-connhelper-ssh"
  }
}
