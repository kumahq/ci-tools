FROM ubuntu:24.04@sha256:c4a8d5503dfb2a3eb8ab5f807da5bc69a85730fb49b5cfca2330194ebcc41c7b

RUN apt update \
  && apt dist-upgrade -y \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    curl \
    dnsutils \
    iproute2 \
    iptables \
    ncat \
    net-tools \
    openssl \
    openssh-server \
    rsync \
    strace \
    tcpdump \
    telnet \
    tmux \
    tzdata \
    vim \
    wget \
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
