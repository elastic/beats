ARG NATS_VERSION=2.0.4
FROM nats:$NATS_VERSION

# build stage
FROM golang:1.21.8 AS build-env
RUN apt-get install git mercurial gcc
RUN git clone https://github.com/nats-io/nats.go.git /nats-go
RUN cd /nats-go/examples/nats-bench && git checkout tags/v1.10.0 && go build .

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
COPY --from=0 / /opt/nats
COPY --from=build-env /nats-go/examples/nats-bench/nats-bench /nats-bench
COPY run.sh /run.sh
# Expose client, management, and cluster ports
EXPOSE 4222 8222 6222
HEALTHCHECK --interval=1s --retries=10 CMD nc -w 1 0.0.0.0 8222 </dev/null
# Run via the configuration file
CMD ["/run.sh"]
