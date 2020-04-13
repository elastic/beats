#!/bin/sh


for i in 1 2 3 4 5;
do
  a=`nslookup host.docker.internal | grep "** server can't find " | wc -l`;
  if [ $a -gt 0 ]; then
    # this works only on Linux envs
    HOST_DOMAIN="0.0.0.0"
  else
    # this works only on Mac envs
    HOST_DOMAIN="host.docker.internal"
    break
  fi
done



REMOTE="$HOST_DOMAIN:9201"

sed -i "s/REMOTE/$REMOTE/g" /etc/prometheus/prometheus.yml

/bin/prometheus --config.file=/etc/prometheus/prometheus.yml \
 --storage.tsdb.path=/prometheus \
 --web.console.libraries=/usr/share/prometheus/console_libraries \
 --web.console.templates=/usr/share/prometheus/consoles
