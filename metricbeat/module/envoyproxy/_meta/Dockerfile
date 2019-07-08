ARG TAG=v1.7.0
FROM envoyproxy/envoy:${TAG}
RUN apt-get update
COPY ./envoy.json /etc/envoy.json
EXPOSE 10000 9901
HEALTHCHECK --interval=1s --retries=90 CMD wget -O - http://localhost:9901/clusters | grep health_flags | grep healthy
CMD /usr/local/bin/envoy -c /etc/envoy.json
