# Tomcat is started to fetch Jolokia metrics from it
FROM golang:1.8.3

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK CMD curl -f curl localhost:8080/

CMD go run main.go
