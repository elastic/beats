# FROM logstash:2
FROM java:8-jre

ENV LOGSTASH_REPO https://github.com/elastic/logstash.git
ENV LUMBERJACK_REPO https://github.com/urso/ruby-lumberjack.git
ENV LUMBERJACK_PLUGIN_REPO https://github.com/urso/logstash-input-lumberjack.git

ENV LUMBERJACK_BRANCH dev/json
ENV LUMBERJACK_PLUGIN_BRANCH beat_testing

ENV PATH $PATH:/opt/logstash/bin:/opt/logstash/vendor/jruby/bin

# add missing packages
RUN set -x && \
    apt-get update -y && \
    apt-get install -y git rbenv rake && \
    rm -rf /var/lib/apt/lists/*

# clone repos + checkout dev branches
RUN set -x && \
    mkdir -p /opt/logstash && \
    git clone $LOGSTASH_REPO /opt/logstash && \
    git clone $LUMBERJACK_REPO /ruby-lumberjack && \
    git clone $LUMBERJACK_PLUGIN_REPO /logstash-input-lumberjack && \
    cd /ruby-lumberjack && \
    git checkout -b $LUMBERJACK_BRANCH origin/$LUMBERJACK_BRANCH && \
    gem build jls-lumberjack.gemspec && \
    cd /logstash-input-lumberjack && \
    git checkout -b $LUMBERJACK_PLUGIN_BRANCH origin/$LUMBERJACK_PLUGIN_BRANCH && \
    gem build logstash-input-lumberjack.gemspec

# install plugins
RUN set -x && \
    echo 'gem "jls-lumberjack", :path => "/ruby-lumberjack"' >> /opt/logstash/Gemfile && \
    cd /opt/logstash && rake test:install-core && \
    cd /logstash-input-lumberjack && \
    plugin install --no-verify logstash-input-lumberjack*.gem && \
    plugin install --no-verify logstash-output-elasticsearch

COPY logstash.conf /logstash.conf
COPY docker-entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

CMD logstash agent -f /logstash.conf
