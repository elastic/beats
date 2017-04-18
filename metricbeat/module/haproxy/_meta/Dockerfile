FROM haproxy:1.6
RUN apt-get update && apt-get install -y netcat
HEALTHCHECK CMD nc -z localhost 14567
COPY ./haproxy.conf /usr/local/etc/haproxy/haproxy.cfg
EXPOSE 14567
