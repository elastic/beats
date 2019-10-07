ARG LOGSTASH_VERSION
FROM docker.elastic.co/logstash/logstash:${LOGSTASH_VERSION}

COPY healthcheck.sh /
ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh
