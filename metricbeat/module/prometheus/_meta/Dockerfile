FROM prom/prometheus:v2.6.0

USER root
RUN mkdir /run # Needed to copy /run/compose_env on startup
USER nobody

HEALTHCHECK --interval=1s --retries=90 CMD nc -w 1 localhost 9090 </dev/null
EXPOSE 9090
