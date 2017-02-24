# TODO: No tags currently exist for this image. Tags should be used whever possible
# as otherwise builds are not identical over time.
FROM spotify/kafka

RUN apt-get update && apt-get install -y netcat
HEALTHCHECK CMD nc -z localhost 9092

EXPOSE 2181 9092

ENV ADVERTISED_HOST kafka
