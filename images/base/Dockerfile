FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Python stays on the base image because Ansible needs it on the target.
# policy-rc.d stops package installs from calling a service manager that is not running.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        openssh-server \
        procps \
        python3 \
        python3-apt \
        sudo \
    && rm -rf /var/lib/apt/lists/* /var/lib/apt/periodic \
    && mkdir -p /var/run/sshd \
    && ssh-keygen -A \
    && printf '%s\n' \
        'PermitRootLogin prohibit-password' \
        'PasswordAuthentication no' \
        'PubkeyAuthentication yes' \
        > /etc/ssh/sshd_config.d/lab.conf \
    && printf '%s\n' '#!/bin/sh' 'exit 101' > /usr/sbin/policy-rc.d \
    && chmod 755 /usr/sbin/policy-rc.d

COPY entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh

EXPOSE 22
ENTRYPOINT ["/entrypoint.sh"]
