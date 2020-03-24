ARG NODE_EXPORTER_VERSION
FROM prom/node-exporter:v${NODE_EXPORTER_VERSION}
EXPOSE 9100
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9100 </dev/null

