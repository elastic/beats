ARG MONGODB_VERSION
FROM mongo:${MONGODB_VERSION}
RUN apt update && yes | apt install netcat-openbsd
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 27017
