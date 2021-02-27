ARG ELASTICSEARCH_VERSION
FROM docker.elastic.co/elasticsearch/elasticsearch:${ELASTICSEARCH_VERSION}
HEALTHCHECK --interval=1s --retries=300 CMD curl -f http://localhost:9200/_license
