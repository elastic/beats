<<<<<<< HEAD
FROM golang:1.23.8
=======
FROM golang:1.24.3
>>>>>>> 9f2953ed7 ([Automation] Bump Golang version to 1.24.3 (#44270))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
