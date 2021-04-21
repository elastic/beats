ARG CONSUL_VERSION
FROM consul:${CONSUL_VERSION}

ENV CONSUL_BIND_INTERFACE='eth0'

EXPOSE 8500

# Wait till the service reports runtime metrics
HEALTHCHECK --interval=1s --retries=90 CMD curl -s http://0.0.0.0:8500/v1/agent/metrics | grep -q consul.runtime
