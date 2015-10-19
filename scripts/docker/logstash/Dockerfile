# FROM logstash:2
FROM java:8-jre

ENV DEB_URL https://download.elastic.co/logstash/logstash/packages/debian/logstash_1.5.4-1_all.deb

ENV PATH $PATH:/opt/logstash/bin:/opt/logstash/vendor/jruby/bin

RUN set -x && \
    mkdir -p /var/tmp && \
    wget -qO /var/tmp/logstash.deb $DEB_URL && \
    apt-get update -y && \
    apt-get install -y logrotate git && \
    dpkg -i /var/tmp/logstash.deb && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ENV LUMBERJACK_REPO https://github.com/elastic/ruby-lumberjack.git
ENV LUMBERJACK_BRANCH fix/small-fixes-for-beats

ENV LUMBERJACK_PLUGIN_REPO https://github.com/ph/logstash-input-beats.git
ENV LUMBERJACK_PLUGIN_BRANCH fix/initial-beats

# 1. clone repos + checkout dev branches
# 2. patch lumberjack server module
# 3. patch lumberjack plugin
RUN set -x && \
    mkdir -p /opt/logstash && \
    git clone $LUMBERJACK_REPO /ruby-lumberjack && \
    git clone $LUMBERJACK_PLUGIN_REPO /logstash-input-beats && \
    cd /ruby-lumberjack && \
    git checkout -b $LUMBERJACK_BRANCH origin/$LUMBERJACK_BRANCH && \
    cd /logstash-input-beats && \
    git checkout -b $LUMBERJACK_PLUGIN_BRANCH origin/$LUMBERJACK_PLUGIN_BRANCH

RUN set -x && \
    rm -fR /opt/logstash/vendor/bundle/jruby/1.9/gems/jls-lumberjack-*/* && \
     cp -R /ruby-lumberjack/spec /ruby-lumberjack/lib /opt/logstash/vendor/bundle/jruby/1.9/gems/jls-lumberjack-*/

RUN set -x && \
     \
     rm -fR /opt/logstash/vendor/bundle/jruby/1.9/gems/logstash-input-lumberjack-*/lib && \
    cp -R /logstash-input-beats/lib /opt/logstash/vendor/bundle/jruby/1.9/gems/logstash-input-lumberjack-*/.

COPY logstash.conf.tmpl /logstash.conf.tmpl
COPY docker-entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

CMD logstash agent -f /logstash.conf

