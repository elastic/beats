#!/usr/bin/env bash

# install logstash
set -x && \
    mkdir -p /var/tmp && \
    wget -qO /var/tmp/logstash.deb $1 && \
    apt-get update -y && \
    apt-get install -y logrotate git && \
    dpkg -i /var/tmp/logstash.deb && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

export PATH=$PATH:/usr/share/logstash/bin
logstash-plugin install logstash-input-beats
