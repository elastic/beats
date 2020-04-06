#!/bin/sh

# This script is used with old and new versions of nats,
# and they use different names for their binaries, detect
# which one is used and run it.

# NATS 2.X
if [ -x /opt/nats/nats-server ]; then
	exec /opt/nats/nats-server -c /opt/nats/nats-server.conf
fi

# NATS 1.X
if [ -x /opt/nats/gnatsd ]; then
	exec /opt/nats/gnatsd -c /opt/nats/gnatsd.conf
fi

echo "Couldn't find the nats server binary"
exit 1
