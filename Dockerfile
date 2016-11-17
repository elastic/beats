FROM golang:1.7.1
MAINTAINER Nicolas Ruflin <ruflin@elastic.co>

RUN set -x && \
    apt-get update && \
    apt-get install -y netcat && \
    apt-get clean

COPY libbeat/scripts/docker-entrypoint.sh /entrypoint.sh

RUN mkdir -p /etc/pki/tls/certs
COPY testing/environments/docker/logstash/pki/tls/certs/logstash.crt /etc/pki/tls/certs/logstash.crt

# Create a copy of the respository inside the container.
COPY . /go/src/github.com/elastic/beats/
