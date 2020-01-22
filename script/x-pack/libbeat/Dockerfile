FROM golang:1.13.6

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip rsync virtualenv && \
    apt-get clean

RUN pip install --upgrade setuptools

# Setup work environment
ENV LIBBEAT_PATH /go/src/github.com/elastic/beats/x-pack/libbeat

RUN mkdir -p $LIBBEAT_PATH/build/coverage
WORKDIR $LIBBEAT_PATH
