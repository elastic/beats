FROM golang:1.13.6
MAINTAINER Noémi Ványi <noemi.vanyi@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
      python-pip virtualenv python3 libsystemd-dev libc6-dev-i386 gcc-arm-linux-gnueabi && \
    apt-get clean

ENV PYTHON_ENV=/tmp/python-env

RUN pip install --upgrade setuptools

# Setup work environment
ENV JOURNALBEAT_PATH /go/src/github.com/elastic/beats/journalbeat

RUN mkdir -p $JOURNALBEAT_PATH/build/coverage
WORKDIR $JOURNALBEAT_PATH
HEALTHCHECK CMD exit 0
