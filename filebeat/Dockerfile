FROM golang:1.7.6
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip virtualenv && \
    apt-get clean

# Setup work environment
ENV FILEBEAT_PATH /go/src/github.com/elastic/beats/filebeat

RUN mkdir -p $FILEBEAT_PATH/build/coverage
WORKDIR $FILEBEAT_PATH
