FROM golang:1.18

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         netcat \
         rsync \
         python3 \
         python3-pip \
         python3-venv \
      && rm -rf /var/lib/apt/lists/*

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2

# Setup work environment
ENV LIBBEAT_PATH /go/src/github.com/elastic/beats/x-pack/libbeat

RUN mkdir -p $LIBBEAT_PATH/build/coverage
WORKDIR $LIBBEAT_PATH
