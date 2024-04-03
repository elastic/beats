ARG STAN_VERSION=0.15.1
FROM nats-streaming:$STAN_VERSION

# build stage
FROM golang:1.21.8 AS build-env
RUN apt-get install git mercurial gcc
RUN git clone https://github.com/nats-io/stan.go.git /stan-go
RUN cd /stan-go/examples/stan-bench && git checkout tags/v0.5.2 && go build .

# create an enhanced container with nc command available since nats is based
# on scratch image making healthcheck impossible
FROM alpine:latest
RUN apk add --no-cache --upgrade bash
COPY --from=0 nats-streaming-server /nats-streaming-server
COPY --from=build-env /stan-go/examples/stan-bench/stan-bench /stan-bench
# Expose client, management, and cluster ports
EXPOSE 4222 8222
ADD healthcheck.sh /healthcheck.sh
RUN ["chmod", "+x", "/healthcheck.sh"]
ADD run.sh /run.sh

# Healthcheck waits until channels have been created by the benchmark that runs inside
HEALTHCHECK --interval=1s --retries=100 CMD /healthcheck.sh

# Run with default memory based store
ENTRYPOINT ["/run.sh"]
