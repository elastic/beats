#!/usr/bin/env bash

# download & install logstash
set -x && \
    cd /opt && \
    wget -q $URL && \
    tar xzf logstash-$VERSION.tar.gz

#logstash-plugin install logstash-input-beats
