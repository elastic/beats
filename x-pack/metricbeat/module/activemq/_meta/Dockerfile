FROM openjdk:8-jdk-alpine3.9

ARG ACTIVEMQ_VERSION

ENV ACTIVEMQ_HOME /opt/activemq
ENV ACTIVEMQ apache-activemq-$ACTIVEMQ_VERSION
ENV ACTIVEMQ_STOMP=61613 ACTIVEMQ_REST=8161
ENV ACTIVEMQ_OPTS="-Dorg.apache.activemq.audit=true"

RUN set -x && \
    mkdir -p /opt && \
    apk --update add --virtual build-dependencies curl netcat-openbsd

RUN curl --fail https://archive.apache.org/dist/activemq/$ACTIVEMQ_VERSION/$ACTIVEMQ-bin.tar.gz -o $ACTIVEMQ-bin.tar.gz

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

HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 -v 127.0.0.1 $ACTIVEMQ_STOMP </dev/null && \
    nc -w 1 -v 127.0.0.1 $ACTIVEMQ_REST </dev/null

CMD ["/bin/sh", "-c", "bin/activemq console"]
