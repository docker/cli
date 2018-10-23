#!/bin/sh
set -ex
mkdir -m 0700 -p /home/penguin/.ssh
chown -R penguin:penguin /home/penguin
/usr/sbin/sshd -E /var/log/sshd.log
exec dockerd-entrypoint.sh $@
