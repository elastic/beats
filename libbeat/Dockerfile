# Beats dockerfile used for testing
FROM golang:1.13.6
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
         netcat python-pip virtualenv python3 libpcap-dev && \
    apt-get clean

ENV PYTHON_ENV=/tmp/python-env

RUN test -d ${PYTHON_ENV} || virtualenv -p python3 ${PYTHON_ENV}
COPY ./tests/system/requirements.txt /tmp/requirements.txt

# Upgrade pip to make sure to have the most recent version
RUN . ${PYTHON_ENV}/bin/activate && pip install -U pip
RUN . ${PYTHON_ENV}/bin/activate && pip install -Ur /tmp/requirements.txt

# Libbeat specific
RUN mkdir -p /etc/pki/tls/certs
