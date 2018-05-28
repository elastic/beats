FROM prom/prometheus:v1.5.1
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9090 </dev/null
EXPOSE 9090
