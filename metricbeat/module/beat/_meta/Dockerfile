FROM docker.elastic.co/beats/metricbeat:7.3.0

COPY healthcheck.sh /
HEALTHCHECK --interval=1s --retries=300 CMD sh /healthcheck.sh

ENTRYPOINT [ "metricbeat", "-E", "http.enabled=true", "-E", "http.host=0.0.0.0" ]
