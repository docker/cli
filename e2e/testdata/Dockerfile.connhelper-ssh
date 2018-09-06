FROM docker:test-dind
RUN apk --no-cache add shadow openssh-server && \
  groupadd -f docker && \
  useradd -m penguin && \
  usermod -aG docker penguin && \
  usermod -p $(head -c32 /dev/urandom | base64) penguin && \
  chsh -s /bin/sh penguin && \
  ssh-keygen -A
# workaround: ssh session excludes /usr/local/bin from $PATH
RUN  ln -s /usr/local/bin/docker /usr/bin/docker
COPY ./connhelper-ssh/entrypoint.sh /
EXPOSE 22
ENTRYPOINT ["/entrypoint.sh"]
# usage: docker run --privileged -e TEST_CONNHELPER_SSH_ID_RSA_PUB=$(cat ~/.ssh/id_rsa.pub) -p 22 $THIS_IMAGE
