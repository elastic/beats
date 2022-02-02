FROM golang:1.17.6

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         libsystemd-dev \
         netcat \
         rsync \
         python3 \
         python3-pip \
         python3-venv \
         libpcap-dev \
      && rm -rf /var/lib/apt/lists/*

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2
