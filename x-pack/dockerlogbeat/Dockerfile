ARG versionString
FROM golang:${versionString} as builder

WORKDIR /go/src/github.com/elastic/beats/x-pack/dockerlogbeat
COPY . ../..

ENV GOPATH=/go
ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dockerlogbeat


FROM alpine:3.7 as final
RUN apk --no-cache add ca-certificates
RUN mkdir /contmount
COPY --from=builder /go/src/github.com/elastic/beats/x-pack/dockerlogbeat/dockerlogbeat /usr/bin/dockerlogbeat
