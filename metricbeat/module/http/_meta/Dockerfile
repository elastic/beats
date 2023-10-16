<<<<<<< HEAD
FROM golang:1.19.12
=======
FROM golang:1.20.10
>>>>>>> 6be0d18448 ([Automation] Bump Golang version to 1.20.10 (#36846))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
