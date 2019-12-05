ARG ENVOYPROXY_VERSION
FROM envoyproxy/envoy:${ENVOYPROXY_VERSION}
RUN apt-get update && apt-get install -y wget
EXPOSE 10000 9901
COPY envoy.yaml /etc/envoy.yaml
HEALTHCHECK --interval=5s --retries=90 CMD wget -O - http://localhost:9901/clusters | grep health_flags | grep healthy
CMD /usr/local/bin/envoy -c /etc/envoy.yaml
