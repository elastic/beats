FROM elasticsearch:2.1.0

ENV ES_USER=beats
ENV ES_PASS=testing

ENV ES_HOME=/usr/share/elasticsearch
ENV PATH=$ES_HOME/bin/shield:$ES_HOME/bin:$PATH

RUN rm -fR /etc/elasticsearch && \
    ln -s $ES_HOME/config /etc/elasticsearch && \
    plugin install license && \
    plugin install shield

COPY docker-entrypoint-shield.sh /

ENTRYPOINT ["/docker-entrypoint-shield.sh"]
