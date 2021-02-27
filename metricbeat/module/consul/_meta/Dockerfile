ARG CONSUL_VERSION
FROM consul:${CONSUL_VERSION}

ENV CONSUL_BIND_INTERFACE='eth0'

EXPOSE 8500

# Use the same healthcheck as the Windows version of the image.
# https://github.com/Microsoft/mssql-docker/blob/a3020afeec9be1eb2d67645ac739438eb8f2c545/windows/mssql-server-windows/dockerfile#L31
HEALTHCHECK --interval=1s --retries=90 CMD curl http://0.0.0.0:8500/v1/agent/metrics
