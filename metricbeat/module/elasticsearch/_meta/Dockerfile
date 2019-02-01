FROM docker.elastic.co/elasticsearch/elasticsearch:6.6.0
HEALTHCHECK --interval=1s --retries=300 CMD curl -f http://localhost:9200/_xpack/license
