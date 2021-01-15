#!/bin/sh

/bin/prometheus --config.file=/etc/prometheus/prometheus.yml \
 --storage.tsdb.path=/prometheus \
 --web.console.libraries=/usr/share/prometheus/console_libraries \
 --web.console.templates=/usr/share/prometheus/consoles
