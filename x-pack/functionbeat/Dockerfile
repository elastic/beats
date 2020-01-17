FROM golang:1.13.6
MAINTAINER Pier-Hugues Pellerin <ph@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip rsync virtualenv python3 && \
    apt-get clean

ENV PYTHON_ENV=/tmp/python-env

RUN pip install --upgrade setuptools

# Setup work environment
ENV FUNCTIONBEAT_PATH /go/src/github.com/elastic/beats/x-pack/functionbeat

RUN mkdir -p $FUNCTIONBEAT_PATH/build/coverage
WORKDIR $FUNCTIONBEAT_PATH
