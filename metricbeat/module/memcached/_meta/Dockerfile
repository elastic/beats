FROM memcached:1.4.35-alpine

USER root
RUN apk update
RUN apk add netcat-openbsd
USER memcache

HEALTHCHECK CMD nc -z localhost 11211
