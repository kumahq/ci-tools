FROM ubuntu:24.04@sha256:9cbed754112939e914291337b5e554b07ad7c392491dba6daf25eef1332a22e8

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
