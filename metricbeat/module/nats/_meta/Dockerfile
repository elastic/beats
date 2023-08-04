ARG NATS_VERSION=2.0.4
FROM nats:$NATS_VERSION

# build stage
FROM golang:1.13-alpine3.11 AS build-env
RUN apk --no-cache add build-base git mercurial gcc
RUN cd src && go get -d github.com/nats-io/nats.go/
RUN cd src/github.com/nats-io/nats.go/examples/nats-bench && git checkout tags/v1.10.0 && go build .

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
COPY --from=0 / /opt/nats
COPY --from=build-env /go/src/github.com/nats-io/nats.go/examples/nats-bench/nats-bench /nats-bench
COPY run.sh /run.sh
# Expose client, management, and cluster ports
EXPOSE 4222 8222 6222
HEALTHCHECK --interval=1s --retries=10 CMD nc -w 1 0.0.0.0 8222 </dev/null
# Run via the configuration file
CMD ["/run.sh"]
