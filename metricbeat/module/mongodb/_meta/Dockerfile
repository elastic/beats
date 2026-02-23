ARG MONGODB_VERSION
FROM mongo:${MONGODB_VERSION}

# remove the mongodb-org.list as the GPG key is expired and we don't need it.
RUN rm /etc/apt/sources.list.d/mongodb-org.list && \
    apt update && \
    yes | apt install netcat-openbsd
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 27017
