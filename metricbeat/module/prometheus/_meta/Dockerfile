ARG PROMETHEUS_VERSION
FROM prom/prometheus:v${PROMETHEUS_VERSION}
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9090 </dev/null
EXPOSE 9090
ADD prometheus.yml /etc/prometheus/prometheus.yml
ADD run.sh /run.sh
RUN ls
ENTRYPOINT ["/run.sh"]
