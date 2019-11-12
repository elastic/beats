ARG BEAT_VERSION
FROM docker.elastic.co/beats/metricbeat:${BEAT_VERSION}

COPY healthcheck.sh /
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh

ENTRYPOINT [ "metricbeat", "-E", "http.enabled=true", "-E", "http.host=0.0.0.0" ]
