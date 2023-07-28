<<<<<<< HEAD
FROM golang:1.19.12
=======
FROM golang:1.20.6
>>>>>>> a278734f11 (upgrade Go to 1.20.6 (#36000))

COPY test/main.go main.go

EXPOSE 8080

HEALTHCHECK --interval=1s --retries=90 CMD curl -f curl localhost:8080/

CMD go run main.go
