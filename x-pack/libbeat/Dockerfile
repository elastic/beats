FROM golang:1.24.9

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         netcat-openbsd \
         rsync \
         python3 \
         python3-pip \
         python3-venv \
         pyenv \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.19.10 uses Debian 12 which now enforces PEP668.
RUN pyenv install 3.9.13
RUN pyenv local 3.9.13

ENV VIRTUAL_ENV=/opt/venv
RUN pyenv exec python -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN pip3 install --upgrade pip==25.1.1

# Setup work environment
ENV LIBBEAT_PATH /go/src/github.com/elastic/beats/x-pack/libbeat

RUN mkdir -p $LIBBEAT_PATH/build/coverage
WORKDIR $LIBBEAT_PATH
