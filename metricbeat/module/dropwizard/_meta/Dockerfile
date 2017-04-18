FROM golang:1.7
RUN apt-get update && apt-get install curl
COPY main.go /go

HEALTHCHECK CMD curl -f http://localhost:9090/metrics/metrics
EXPOSE 9090

CMD go run /go/main.go
