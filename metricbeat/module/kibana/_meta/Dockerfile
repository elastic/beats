FROM docker.elastic.co/kibana/kibana:6.0.0-beta1
HEALTHCHECK --interval=1s --retries=120 CMD curl -f http://localhost:5601/api/status | grep '"disconnects"'
