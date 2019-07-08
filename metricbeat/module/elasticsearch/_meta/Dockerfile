ARG TAG=7.0.0
FROM docker.elastic.co/elasticsearch/elasticsearch:${TAG}
HEALTHCHECK --interval=1s --retries=300 CMD curl -f http://localhost:9200/_license
