ARG HAPROXY_VERSION
FROM haproxy:${HAPROXY_VERSION}
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 14567
COPY ./haproxy.conf /usr/local/etc/haproxy/haproxy.cfg
EXPOSE 14567
