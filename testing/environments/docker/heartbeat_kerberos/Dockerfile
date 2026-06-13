# Self-contained Kerberos test fixture for Heartbeat's HTTP monitor SPNEGO
# (Negotiate) authentication. It runs an MIT KDC and a small SPNEGO-protected
# HTTP server in one container, following the same "server-in-container, client
# uses password auth + committed krb5.conf" pattern as the elasticsearch_kerberos
# environment.
#
# Principals:
#   testuser@EXAMPLE.COM        password: testpass   (client, password auth)
#   HTTP/localhost@EXAMPLE.COM  keytab at /etc/http.keytab (service)

# Stage 1: build the SPNEGO HTTP server.
FROM golang:1.26.4 AS builder
WORKDIR /src
COPY spnego-server/go.mod spnego-server/go.sum ./
RUN go mod download
COPY spnego-server/main.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /spnego-server .

# Stage 2: KDC + SPNEGO server.
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install -y --no-install-recommends krb5-kdc krb5-admin-server krb5-user \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /spnego-server /usr/local/bin/spnego-server
COPY setup.sh /setup.sh
RUN chmod +x /setup.sh

# 88: KDC, 8080: SPNEGO-protected HTTP server.
EXPOSE 88/tcp 88/udp 8080/tcp

CMD ["/setup.sh"]
