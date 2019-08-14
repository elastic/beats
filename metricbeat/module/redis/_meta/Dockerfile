ARG REDIS_VERSION=3.2.12
FROM redis:${REDIS_VERSION}-alpine
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 6379
