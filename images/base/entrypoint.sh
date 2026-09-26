#!/bin/sh
set -eu

if [ -f /bootstrap/authorized_keys ]; then
  install -d -m 700 /root/.ssh
  install -m 600 /bootstrap/authorized_keys /root/.ssh/authorized_keys
fi

exec /usr/sbin/sshd -D -e
