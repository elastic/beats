FROM docker.elastic.co/logstash/logstash:7.3.0

COPY healthcheck.sh /
ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh
