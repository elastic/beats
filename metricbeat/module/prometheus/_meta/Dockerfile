ARG PROMETHEUS_VERSION
FROM prom/prometheus:v${PROMETHEUS_VERSION}
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9090 </dev/null
EXPOSE 9090
