ARG ENGINE_VERSION
# ENGINE_SOURCE is "dind" or "moby-bin" (moby/moby-bin:master).
ARG ENGINE_SOURCE=dind

FROM docker:${ENGINE_VERSION}-dind AS engine-dind

# moby/moby-bin only has the daemon binaries.
FROM docker:dind AS engine-moby-bin
COPY --link --from=moby/moby-bin:master / /usr/local/bin/

FROM engine-${ENGINE_SOURCE}

# Trust the tlsregistry CA so dockerd connects without --insecure-registry.
COPY registry/certs/ca.crt /usr/local/share/ca-certificates/tlsregistry-ca.crt
RUN update-ca-certificates
