ARG STAN_VERSION=0.15.1
FROM nats-streaming:$STAN_VERSION

# build stage
FROM golang:1.13-alpine AS build-env
RUN apk --no-cache add build-base git bzr mercurial gcc
RUN cd src && go get github.com/nats-io/stan.go/
RUN cd src/github.com/nats-io/stan.go/examples/stan-bench && git checkout tags/v0.5.2 && go build .

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
RUN apk add --no-cache --upgrade bash
COPY --from=0 nats-streaming-server /nats-streaming-server
COPY --from=build-env /go/src/github.com/nats-io/stan.go/examples/stan-bench/stan-bench /stan-bench
# Expose client, management, and cluster ports
EXPOSE 4222 8222
HEALTHCHECK --interval=1s --retries=10 CMD nc -w 1 0.0.0.0 8222 </dev/null
ADD run.sh /run.sh
# Run with default memory based store
ENTRYPOINT ["/run.sh"]
