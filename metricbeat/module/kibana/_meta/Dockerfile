ARG KIBANA_VERSION
FROM docker.elastic.co/kibana/kibana:${KIBANA_VERSION}
RUN curl https://stedolan.github.io/jq/download/linux64/jq > /usr/share/kibana/jq
RUN chmod +x jq
HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD if [ $(curl -s http://myelastic:changeme@localhost:5601/api/status > status && cat status | ./jq ".status.overall.state") == '"green"' ]; then exit 0; else exit 1; fi
