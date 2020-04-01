FROM golang:1.13.9

ENV DEBIAN_FRONTEND noninteractive

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         netcat \
         krb5-user \ 
         libpcap-dev \
         python3 \
         python3-pip \
         python3-venv \
      && rm -rf /var/lib/apt/lists/*

ENV PYTHON_ENV=/tmp/python-env

RUN pip3 install --upgrade pip
RUN pip3 install --upgrade setuptools
RUN pip3 install --upgrade docker-compose==1.23.2
ADD kafka.keytab /etc/security/keytabs/kafka.keytab 
ADD krb5.conf /etc/krb5.conf

# Libbeat specific
RUN mkdir -p /etc/pki/tls/certs
