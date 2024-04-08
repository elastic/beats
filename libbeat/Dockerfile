FROM golang:1.21.9

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         netcat-openbsd \
         libpcap-dev \
         python3 \
         python3-pip \
         python3-venv \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.19.10 uses Debian 12 which now enforces PEP668.
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN pip3 install --upgrade pip==20.1.1

# Libbeat specific
RUN mkdir -p /etc/pki/tls/certs
