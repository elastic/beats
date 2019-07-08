ARG TAG=7.0.0
FROM docker.elastic.co/logstash/logstash:${TAG}

COPY healthcheck.sh /
ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh
