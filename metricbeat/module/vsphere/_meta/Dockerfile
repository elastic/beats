ARG VSPHERE_GOLANG_VERSION
FROM golang:${VSPHERE_GOLANG_VERSION}-alpine

RUN apk add --no-cache curl git
RUN go install github.com/vmware/govmomi/vcsim@latest

HEALTHCHECK --interval=1s --retries=60 --timeout=10s CMD curl http://localhost:8989/
CMD vcsim -l :8989
