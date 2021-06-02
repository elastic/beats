FROM golang:1.16.4

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
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
ENV FUNCTIONBEAT_PATH /go/src/github.com/elastic/beats/x-pack/functionbeat

RUN mkdir -p $FUNCTIONBEAT_PATH/build/coverage
WORKDIR $FUNCTIONBEAT_PATH
