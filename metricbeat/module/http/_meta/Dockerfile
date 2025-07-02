<<<<<<< HEAD
FROM golang:1.24.5
=======
FROM golang:1.24.3
>>>>>>> 15a959f97 ([docs][metricbeat] - remove asciidoc and add markdown (#44947))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
