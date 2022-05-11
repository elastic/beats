FROM golang:1.17.9

RUN \
    apt update \
      && DEBIAN_FRONTEND=noninteractive apt-get install -qq -y --no-install-recommends \
         netcat \
         python3 \
         python3-dev \
         python3-pip \
         python3-venv \
      && rm -rf /var/lib/apt/lists/*

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2

# Add healthcheck for the docker/healthcheck metricset to check during testing.
HEALTHCHECK CMD exit 0
