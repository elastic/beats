# Beats dockerfile used for testing
FROM golang:1.7
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y netcat python-virtualenv python-pip && \
    apt-get clean

ENV PYTHON_ENV=/tmp/python-env

RUN test -d ${PYTHON_ENV} || virtualenv ${PYTHON_ENV}
RUN . ${PYTHON_ENV}/bin/activate && pip install nose jinja2 PyYAML nose-timer

# Packetbeat specifics
RUN apt-get install -y libpcap-dev geoip-database && apt-get clean

