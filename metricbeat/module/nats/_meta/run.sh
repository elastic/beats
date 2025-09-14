#!/bin/sh

# NATS 2.X
if [ -x /opt/nats/nats-server ]; then
    if [[ -z "${ROUTES}" ]]; then
        (/opt/nats/nats-server --cluster nats://0.0.0.0:6222 --http_port 8222 --port 4222 -js --server_name test1 --cluster_name test --routes nats://nats-routes:6222) &
    else
        (/opt/nats/nats-server --cluster nats://0.0.0.0:6222 --http_port 8222 --port 4222 -js --server_name test2 --cluster_name test --routes nats://nats:6222) &
    fi

    /setup.sh
else
    echo "Couldn't find the nats server binary"
    exit 1
fi

