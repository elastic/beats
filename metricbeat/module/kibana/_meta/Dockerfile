ARG KIBANA_VERSION
FROM docker.elastic.co/kibana/kibana:${KIBANA_VERSION}
HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD curl -u myelastic:changeme -f "http://localhost:5601/api/stats?extended=true&legacy=true&exclude_usage=false" | grep '"status":"green"'
