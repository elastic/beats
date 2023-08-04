<<<<<<< HEAD
FROM golang:1.16.5
=======
FROM golang:1.20.7
>>>>>>> a88f111fc6 ([Automation] Bump Golang version to 1.20.7 (#36241))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
