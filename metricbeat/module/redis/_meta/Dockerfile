ARG TAG=3.2.12-alpine
FROM redis:${TAG}
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 6379
