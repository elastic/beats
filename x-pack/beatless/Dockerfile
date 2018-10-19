FROM golang:1.10.3
MAINTAINER Pier-Hugues Pellerin <ph@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip rsync virtualenv && \
    apt-get clean

RUN pip install --upgrade setuptools

# Setup work environment
ENV BEATLESS_PATH /go/src/github.com/elastic/beats/x-pack/beatless

RUN mkdir -p $BEATLESS_PATH/build/coverage
WORKDIR $BEATLESS_PATH
