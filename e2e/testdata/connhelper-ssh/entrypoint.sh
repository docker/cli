#!/bin/sh
set -ex
mkdir -m 0700 -p /home/penguin/.ssh
echo ${TEST_CONNHELPER_SSH_ID_RSA_PUB} > /home/penguin/.ssh/authorized_keys
chmod 0600 /home/penguin/.ssh/authorized_keys
chown -R penguin:penguin /home/penguin
/usr/sbin/sshd -E /var/log/sshd.log
exec dockerd-entrypoint.sh $@
