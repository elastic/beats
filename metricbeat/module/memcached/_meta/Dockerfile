FROM memcached:1.4.35-alpine

USER root
RUN apk update
RUN apk add netcat-openbsd
USER memcache

HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 11211
