FROM golang:1.21.10

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         libsystemd-dev \
         netcat-openbsd \
         rsync \
         python3 \
         python3-pip \
         python3-venv \
         libpcap-dev \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.19.10 uses Debian 12 which now enforces PEP668.
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade docker-compose==1.23.2
RUN pip3 install --upgrade PyYAML==5.3.1
