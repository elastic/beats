FROM golang:1.13.6

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         python-pip \
         virtualenv \
         python3 \
         librpm-dev \
      && rm -rf /var/lib/apt/lists/*

ENV PYTHON_ENV=/tmp/python-env

RUN pip install --upgrade pip
RUN pip install --upgrade setuptools
RUN pip install --upgrade docker-compose==1.23.2
