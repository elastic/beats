FROM spotify/kafka
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK CMD nc -z localhost 9092
