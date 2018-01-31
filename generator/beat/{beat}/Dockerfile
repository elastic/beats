FROM golang:1.9.2
MAINTAINER {full_name}

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip virtualenv && \
    apt-get clean

RUN pip install --upgrade setuptools

# Setup work environment
ENV APP_PATH /go/src/github.com/{beat_path}

RUN mkdir -p $APP_PATH/build/coverage
WORKDIR $APP_PATH

# Add healthcheck for docker/healthcheck {beat} to check during testing
HEALTHCHECK CMD exit 0
