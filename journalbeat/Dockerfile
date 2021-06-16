FROM golang:1.16.6

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         libsystemd-dev \
         libc6-dev-i386 \
         gcc-arm-linux-gnueabi \
         python3 \
         python3-pip \
         python3-venv \
      && rm -rf /var/lib/apt/lists/*

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2

# Setup work environment
ENV JOURNALBEAT_PATH /go/src/github.com/elastic/beats/journalbeat

RUN mkdir -p $JOURNALBEAT_PATH/build/coverage
WORKDIR $JOURNALBEAT_PATH
HEALTHCHECK CMD exit 0
