FROM docker.elastic.co/logstash/logstash:6.2.4

COPY healthcheck.sh /
ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh
