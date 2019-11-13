FROM golang:1.13.4
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip virtualenv && \
    apt-get clean

RUN pip install --upgrade pip
RUN pip install --upgrade setuptools
RUN pip install --upgrade docker-compose==1.23.2

# Setup work environment
ENV HEARTBEAT_PATH /go/src/github.com/elastic/beats/heartbeat

RUN mkdir -p $HEARTBEAT_PATH/build/coverage
WORKDIR $HEARTBEAT_PATH

# Add healthcheck for docker/healthcheck metricset to check during testing
HEALTHCHECK CMD exit 0
