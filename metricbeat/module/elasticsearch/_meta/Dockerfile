FROM docker.elastic.co/elasticsearch/elasticsearch:6.5.1
HEALTHCHECK --interval=1s --retries=300 CMD curl -f http://localhost:9200/_xpack/license
