ARG TAG=3.9.0
FROM aerospike:${TAG}

RUN apt-get update && apt-get install -y netcat
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 3000
