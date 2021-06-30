ARG NODE_EXPORTER_VERSION
FROM prom/node-exporter:v${NODE_EXPORTER_VERSION}
EXPOSE 9100
HEALTHCHECK --interval=1s --retries=90 CMD wget -q http://localhost:9100/metrics -O - | grep "node_cpu_seconds_total"

