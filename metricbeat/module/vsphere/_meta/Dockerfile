ARG VSPHERE_GOLANG_VERSION
FROM golang:${VSPHERE_GOLANG_VERSION}-alpine

RUN apk add --no-cache git
RUN go get -u github.com/vmware/govmomi/vcsim
CMD vcsim -l :8989
