FROM docker.elastic.co/logstash/logstash:6.2.3

COPY healthcheck.sh /
ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=90 CMD sh /healthcheck.sh
