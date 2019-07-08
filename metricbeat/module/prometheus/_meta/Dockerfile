ARG TAG=v2.6.0
FROM prom/prometheus:${TAG}
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9090 </dev/null
EXPOSE 9090
