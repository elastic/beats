FROM golang:1.24.10

RUN \
    apt-get update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
         python3 \
         python3-pip \
         python3-venv \
         librpm-dev \
         iproute2 \
         pyenv \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.20.6 uses Debian 12 which now enforces PEP668.
RUN pyenv install 3.9.13
RUN pyenv local 3.9.13

ENV VIRTUAL_ENV=/opt/venv
RUN pyenv exec python -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN pip3 install --upgrade pip==25.1.1
