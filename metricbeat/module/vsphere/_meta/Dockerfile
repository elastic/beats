ARG VSPHERE_GOLANG_VERSION
<<<<<<< HEAD
FROM golang:${VSPHERE_GOLANG_VERSION}-alpine
=======
FROM golang:1.20.7
>>>>>>> a88f111fc6 ([Automation] Bump Golang version to 1.20.7 (#36241))

RUN apk add --no-cache curl git
RUN go install github.com/vmware/govmomi/vcsim@v0.30.4

HEALTHCHECK --interval=1s --retries=60 --timeout=10s CMD curl http://localhost:8989/
CMD vcsim -l :8989
