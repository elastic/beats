FROM java:8-jre

RUN apt-get update && \
    apt-get install -y netcat

ARG DOWNLOAD_URL
ARG ELASTIC_VERSION

ENV URL ${DOWNLOAD_URL}/logstash/logstash-${ELASTIC_VERSION}.tar.gz
ENV PATH $PATH:/opt/logstash-${ELASTIC_VERSION}/bin

# Cache variable can be set during building to invalidate the build cache with `--build-arg CACHE=$(date +%s) .`
ARG CACHE=1

# As all snapshot builds have the same url, the image is cached. The date at then can be used to invalidate the image
RUN set -x && \
    cd /opt && \
    wget -qO logstash.tar.gz $URL?${CACHE} && \
    tar xzf logstash.tar.gz


COPY logstash.conf.tmpl /logstash.conf.tmpl
COPY docker-entrypoint.sh /entrypoint.sh

COPY pki /etc/pki

HEALTHCHECK CMD nc -z localhost 5044

ENTRYPOINT ["/entrypoint.sh"]

EXPOSE 5044 5055 9600

CMD logstash -f /logstash.conf --log.level=debug --config.debug --http.host=0.0.0.0
