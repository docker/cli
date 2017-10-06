FROM    docker/compose:1.15.0

RUN     apk add -U bash curl

ARG     DOCKER_CHANNEL=edge
ARG     DOCKER_VERSION=17.06.0-ce
RUN     export URL=https://download.docker.com/linux/static; \
        curl -Ls $URL/$DOCKER_CHANNEL/x86_64/docker-$DOCKER_VERSION.tgz | \
        tar -xz docker/docker && \
        mv docker/docker /usr/local/bin/ && \
        rmdir docker
ENV     DISABLE_WARN_OUTSIDE_CONTAINER=1
WORKDIR /work
COPY    scripts/test/e2e scripts/test/e2e
COPY    e2e/compose-env.yaml e2e/compose-env.yaml

ENTRYPOINT ["bash", "/work/scripts/test/e2e/run"]
