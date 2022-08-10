ARG ZOOKEEPER_VERSION
FROM zookeeper:${ZOOKEEPER_VERSION}
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 2181
