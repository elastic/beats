FROM docker.elastic.co/logstash/logstash:6.0.0-rc2

ENV XPACK_MONITORING_ENABLED=FALSE
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:9600
