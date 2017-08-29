FROM mongo:3.4
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 27017
