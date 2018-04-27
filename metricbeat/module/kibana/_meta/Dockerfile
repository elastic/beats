FROM docker.elastic.co/kibana/kibana:6.2.4
HEALTHCHECK --interval=1s --retries=600 CMD curl -f http://localhost:5601/api/status | grep '"disconnects"'
