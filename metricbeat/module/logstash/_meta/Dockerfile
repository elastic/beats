ARG LOGSTASH_VERSION
FROM docker.elastic.co/logstash/logstash:${LOGSTASH_VERSION}

COPY healthcheck.sh /
COPY pipeline/logstash.conf /usr/share/logstash/pipeline/logstash.conf

ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh
