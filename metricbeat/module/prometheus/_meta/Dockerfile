FROM prom/prometheus:v1.5.1
RUN mkdir /run # Needed to copy /run/compose_env on startup
HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9090 </dev/null
EXPOSE 9090
