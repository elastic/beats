FROM docker.elastic.co/kibana/kibana:6.5.4
HEALTHCHECK --interval=1s --retries=300 CMD curl -f http://localhost:5601/api/status | grep '"disconnects"'
