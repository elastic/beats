<<<<<<< HEAD
FROM golang:1.22.8
=======
FROM golang:1.22.10
>>>>>>> 3cdd8564f ([Automation] Bump Golang version to 1.22.10 (#42095))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
