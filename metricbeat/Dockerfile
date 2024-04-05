FROM golang:1.21.9

RUN \
    apt update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -qq -y --no-install-recommends \
         netcat-openbsd \
         python3 \
         python3-dev \
         python3-pip \
         python3-venv \
      && rm -rf /var/lib/apt/lists/*

# Use a virtualenv to avoid the PEP668 "externally managed environment" error caused by conflicts
# with the system Python installation. golang:1.20.6 uses Debian 12 which now enforces PEP668.
ENV VIRTUAL_ENV=/opt/venv
RUN python3 -m venv $VIRTUAL_ENV
ENV PATH="$VIRTUAL_ENV/bin:$PATH"

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade docker-compose==1.23.2
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade PyYAML==5.3.1

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
