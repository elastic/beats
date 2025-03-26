<<<<<<< HEAD
FROM golang:1.23.8
=======
FROM golang:1.24.1
>>>>>>> 33da6ddb8 ([Automation] Bump Golang version to 1.24.1 (#43488))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
