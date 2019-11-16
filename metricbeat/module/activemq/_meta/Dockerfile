FROM openjdk:8-jre-alpine

ARG ACTIVEMQ_VERSION

ENV ACTIVEMQ_HOME /opt/activemq
ENV ACTIVEMQ apache-activemq-$ACTIVEMQ_VERSION
ENV ACTIVEMQ_STOMP=61613 ACTIVEMQ_REST=8161

RUN set -x && \
    mkdir -p /opt && \
    apk --update add --virtual build-dependencies curl && \
    curl https://archive.apache.org/dist/activemq/$ACTIVEMQ_VERSION/$ACTIVEMQ-bin.tar.gz -o $ACTIVEMQ-bin.tar.gz

RUN tar xvzf $ACTIVEMQ-bin.tar.gz -C  /opt && \
    ln -s /opt/$ACTIVEMQ $ACTIVEMQ_HOME && \
    addgroup -S activemq && adduser -S -H -G activemq -h $ACTIVEMQ_HOME activemq && \
    chown -R activemq:activemq /opt/$ACTIVEMQ && \
    chown -h activemq:activemq $ACTIVEMQ_HOME && \
    apk del build-dependencies && \
    rm -rf /var/cache/apk/* && \
    rm $ACTIVEMQ-bin.tar.gz

USER activemq

WORKDIR $ACTIVEMQ_HOME

EXPOSE $ACTIVEMQ_STOMP $ACTIVEMQ_REST

CMD ["/bin/sh", "-c", "bin/activemq console"]
