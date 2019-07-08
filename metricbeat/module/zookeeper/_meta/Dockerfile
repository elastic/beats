ARG TAG=3.4.8
FROM jplock/zookeeper:${TAG}
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 2181
