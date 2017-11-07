#!/usr/bin/env bash

# setup logstash dependencies
set -x && \
    apt-get update && \
    apt-get install -y --no-install-recommends logrotate && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# download & install logstash
set -x && \
    mkdir -p /var/tmp && \
    wget -qO /var/tmp/logstash.deb $1 && \
    dpkg -i /var/tmp/logstash.deb && \


export PATH=$PATH:/usr/share/logstash/bin
logstash-plugin install logstash-input-beats
