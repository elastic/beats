FROM ubuntu:16.04

RUN apt-get update && \
    apt-get install -y munin-node netcat && \
    apt-get clean && rm rm -rf /var/lib/apt/lists/*

EXPOSE 4949

COPY munin-node.conf /etc/munin/munin-node.conf

HEALTHCHECK --interval=1s --retries=90 CMD nc -z 127.0.0.1 4949

CMD munin-node
