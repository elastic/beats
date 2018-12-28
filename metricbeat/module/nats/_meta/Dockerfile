ARG NATS_VERSION=1.3.0
FROM nats:$NATS_VERSION
HEALTHCHECK --interval=1s --retries=90 CMD curl -o -I -L -s -w "%{http_code}" http://localhost:8222 | grep 200
