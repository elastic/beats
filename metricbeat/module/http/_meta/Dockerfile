<<<<<<< HEAD
FROM golang:1.24.4
=======
FROM golang:1.24.5
>>>>>>> 7e23e08a1 ([Automation] Bump Golang version to 1.24.5 (#45403))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
