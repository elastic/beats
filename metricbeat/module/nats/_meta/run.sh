#!/bin/sh

# This script is used with old and new versions of nats,
# and they use different names for their binaries, detect
# which one is used and run it.

# NATS 2.X
if [ -x /opt/nats/nats-server ]; then
    if [[ -z "${ROUTES}" ]]; then
        (/opt/nats/nats-server --cluster nats://0.0.0.0:6222 --http_port 8222 --port 4222) &
    else
        (/opt/nats/nats-server --cluster nats://0.0.0.0:6222 --http_port 8222 --port 4222 --routes nats://nats:6222) &
    fi
    while true; do /nats-bench -np 1 -n 100000000 -ms 16 foo; done
fi

# NATS 1.X
if [ -x /opt/nats/gnatsd ]; then
    if [[ -z "${ROUTES}" ]]; then
        (/opt/nats/gnatsd --cluster nats://0.0.0.0:6222 --http_port 8222 --port 4222) &
    else
        (/opt/nats/gnatsd --cluster nats://0.0.0.0:6222 --http_port 8222 --port 4222 --routes nats://nats:6222) &
    fi
    while true; do /nats-bench -np 1 -n 100000000 -ms 16 foo; done
fi

echo "Couldn't find the nats server binary"
exit 1
