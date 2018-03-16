FROM golang:1.9.2
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip virtualenv && \
    apt-get clean

RUN pip install --upgrade setuptools

# Setup work environment
ENV AUDITBEAT_PATH /go/src/github.com/elastic/beats/auditbeat

RUN mkdir -p $AUDITBEAT_PATH/build/coverage
WORKDIR $AUDITBEAT_PATH
HEALTHCHECK CMD exit 0
