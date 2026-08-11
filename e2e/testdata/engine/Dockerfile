ARG ENGINE_VERSION
FROM docker:${ENGINE_VERSION}-dind

# Trust the tlsregistry CA so dockerd connects without --insecure-registry.
COPY registry/certs/ca.crt /usr/local/share/ca-certificates/tlsregistry-ca.crt
RUN update-ca-certificates
