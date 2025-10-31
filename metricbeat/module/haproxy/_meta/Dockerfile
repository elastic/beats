ARG HAPROXY_VERSION
FROM haproxy:${HAPROXY_VERSION}
RUN apt-get update && apt-get install -y netcat-openbsd

HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 14567 && nc -z localhost 14570

COPY ./haproxy.conf /usr/local/etc/haproxy/haproxy.cfg
COPY ./certs/server.pem  /usr/local/etc/haproxy/server.pem
COPY ./certs/ca.crt  /usr/local/etc/haproxy/ca.crt

EXPOSE 14567 14568 14569 14570
