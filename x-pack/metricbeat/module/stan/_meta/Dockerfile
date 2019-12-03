ARG STAN_VERSION=0.15.1
FROM nats-streaming:$STAN_VERSION

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
COPY --from=0 /gnatsd /gnatsd
COPY --from=0 gnatsd.conf gnatsd.conf
# Expose client, management, and cluster ports
EXPOSE 4222 8222 6222
HEALTHCHECK --interval=1s --retries=10 CMD nc -w 1 0.0.0.0 8222 </dev/null
# Run via the configuration file
ENTRYPOINT ["/gnatsd"]
CMD ["-c", "gnatsd.conf"]
