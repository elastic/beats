FROM golang:1.8.3
MAINTAINER Steffen Siering <steffen.siering@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip virtualenv && \
    apt-get clean

# Setup work environment
ENV HEARTBEAT_PATH /go/src/github.com/elastic/beats/heartbeat

RUN mkdir -p $HEARTBEAT_PATH/build/coverage
WORKDIR $HEARTBEAT_PATH
