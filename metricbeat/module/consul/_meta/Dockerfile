FROM consul:1.4.2

ENV CONSUL_BIND_INTERFACE='eth0'

EXPOSE 8500

HEALTHCHECK --interval=1s --retries=90 --start-period=10 CMD curl http://0.0.0.0:8500/v1/agent/metrics
