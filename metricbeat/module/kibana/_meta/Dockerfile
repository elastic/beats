ARG KIBANA_VERSION
FROM docker.elastic.co/kibana/kibana:${KIBANA_VERSION}
HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD curl -s http://localhost:5601/api/status?v8format=true | grep -q '"overall":{"level":"available"'
