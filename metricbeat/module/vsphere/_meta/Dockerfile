ARG VSPHERE_GOLANG_VERSION
<<<<<<< HEAD
FROM golang:1.24.9-bookworm
=======
FROM golang:1.24.11-bookworm
>>>>>>> 64983371a ([Automation] Bump Golang version to 1.24.11 (#47903))

RUN apt-get install curl git
RUN go install github.com/vmware/govmomi/vcsim@v0.30.4

HEALTHCHECK --interval=1s --retries=60 --timeout=10s CMD curl http://localhost:8989/
CMD vcsim -l :8989
