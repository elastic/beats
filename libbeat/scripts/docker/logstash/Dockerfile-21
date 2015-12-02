# FROM logstash:2
FROM java:8-jre

ENV LS_VERSION 2

ENV DEB_URL https://download.elastic.co/logstash/logstash/packages/debian/logstash_2.1.0-1_all.deb

ENV PATH $PATH:/opt/logstash/bin:/opt/logstash/vendor/jruby/bin

# install logstash
RUN set -x && \
    mkdir -p /var/tmp && \
    wget -qO /var/tmp/logstash.deb $DEB_URL && \
    apt-get update -y && \
    apt-get install -y logrotate git && \
    dpkg -i /var/tmp/logstash.deb && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN plugin install logstash-input-beats

#COPY logstash.conf.1.5.tmpl /logstash.conf.1.5.tmpl
COPY logstash.conf.2.tmpl /logstash.conf.2.tmpl
COPY docker-entrypoint.sh /entrypoint.sh

COPY pki /etc/pki

ENTRYPOINT ["/entrypoint.sh"]

CMD logstash agent -f /logstash.conf
