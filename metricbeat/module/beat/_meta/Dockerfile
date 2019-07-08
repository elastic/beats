ARG TAG=7.0.0
FROM docker.elastic.co/beats/metricbeat:${TAG}

COPY healthcheck.sh /
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh

ENTRYPOINT [ "metricbeat", "-E", "http.enabled=true", "-E", "http.host=0.0.0.0" ]
