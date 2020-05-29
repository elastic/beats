ARG NATS_VERSION=2.0.4
FROM nats:$NATS_VERSION

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
COPY --from=0 / /opt/nats
COPY run.sh /run.sh
# Expose client, management, and cluster ports
EXPOSE 4222 8222 6222
HEALTHCHECK --interval=1s --retries=10 CMD nc -w 1 0.0.0.0 8222 </dev/null
# Run via the configuration file
CMD ["/run.sh"]
