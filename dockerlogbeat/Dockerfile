FROM  golang:latest as builder

WORKDIR /go/src/github.com/elastic/beats/dockerlogbeat
COPY . .

ENV GOPATH=/go
ARG GOOS=linux
ARG GOARCH=amd64
ARG GOARM=

RUN ls
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hellologdriver


FROM alpine:3.7 as final
RUN apk --no-cache add ca-certificates
RUN mkdir /contmount
COPY --from=builder /go/src/github.com/elastic/beats/dockerlogbeat/hellologdriver /usr/bin/hellologdriver
COPY --from=builder --chown=root:root /go/src/github.com/elastic/beats/dockerlogbeat/beat.yml /usr/bin/beat.yml
COPY --from=builder --chown=root:root /go/src/github.com/elastic/beats/dockerlogbeat/beat.yml /beat.yml
