FROM java:8-jre

ARG LS_DOWNLOAD_URL
ARG LS_VERSION

ENV URL ${LS_DOWNLOAD_URL}/logstash-${LS_VERSION}.tar.gz
ENV PATH $PATH:/opt/logstash-${LS_VERSION}/bin

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

ENTRYPOINT ["/entrypoint.sh"]

EXPOSE 5044 5055

CMD logstash -f /logstash.conf --log.level=debug --config.debug
