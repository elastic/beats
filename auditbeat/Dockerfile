FROM golang:1.12.4

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         python-pip \
         virtualenv \
         librpm-dev \
      && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip
RUN pip install --upgrade setuptools
RUN pip install --upgrade docker-compose==1.23.2
