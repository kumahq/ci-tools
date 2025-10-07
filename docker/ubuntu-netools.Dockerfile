FROM ubuntu:24.04@sha256:728785b59223d755e3e5c5af178fab1be7031f3522c5ccd7a0b32b80d8248123

RUN apt update \
  && apt dist-upgrade -y \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    curl \
    dnsutils \
    iproute2 \
    iptables \
    ncat \
    net-tools \
    openssh-server \
    rsync \
    strace \
    tcpdump \
    telnet \
    tmux \
    tzdata \
    vim \
  && apt clean \
  && rm -rf /var/lib/apt/lists/*

RUN ssh-keygen -A \
  && sed -i s/#PermitRootLogin.*/PermitRootLogin\ yes/ /etc/ssh/sshd_config \
  && sed -i s/#PermitEmptyPasswords.*/PermitEmptyPasswords\ yes/ /etc/ssh/sshd_config \
  && mkdir /var/run/sshd \
  && passwd -d root \
  && chmod a+rwx /root

# do not detach (-D), log to stderr (-e)
CMD ["/usr/sbin/sshd", "-D", "-e"]
