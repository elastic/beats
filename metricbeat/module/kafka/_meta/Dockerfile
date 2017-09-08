# TODO: No tags currently exist for this image. Tags should be used whever possible
# as otherwise builds are not identical over time.
FROM spotify/kafka

HEALTHCHECK --interval=1s --retries=90 CMD /opt/kafka_*/bin/kafka-topics.sh --zookeeper localhost:2181 --list

EXPOSE 2181 9092

ENV ADVERTISED_HOST kafka
