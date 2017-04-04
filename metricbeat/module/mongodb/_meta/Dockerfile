FROM mongo:3.4
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK CMD nc -z localhost 27017
