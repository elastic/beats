FROM golang:1.13.7

RUN \
    apt-get update \
      && apt-get install -y --no-install-recommends \
         python3 \
         python3-pip \
         python3-venv \
         librpm-dev \
         netcat \
         libpcap-dev \
      && rm -rf /var/lib/apt/lists/*

ENV PYTHON_ENV=/tmp/python-env

RUN pip3 install --upgrade pip==20.1.1
RUN pip3 install --upgrade setuptools==47.3.2
RUN pip3 install --upgrade docker-compose==1.23.2
