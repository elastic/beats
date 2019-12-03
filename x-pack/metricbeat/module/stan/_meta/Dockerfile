ARG STAN_VERSION=0.15.1
FROM nats-streaming:$STAN_VERSION

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
COPY --from=0 nats-streaming-server /nats-streaming-server
# Expose client, management, and cluster ports
EXPOSE 4222 8222
HEALTHCHECK --interval=1s --retries=10 CMD nc -w 1 0.0.0.0 8222 </dev/null
# Run with default memory based store
ENTRYPOINT ["/nats-streaming-server"]
CMD ["-m", "8222"]
