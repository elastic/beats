#!/bin/sh

# These commands are compatible with v0.2.1

until /nats server check jetstream 2>&1 | grep -q "OK JetStream"; do
    sleep 1
done

echo "JetStream is ready. Setting up streams and consumers"

# Create streams
/nats stream create stream1 --defaults --subjects="stream1.>"
/nats stream create stream2 --defaults --subjects="stream2.>"

# Create some consumers
/nats consumer create --defaults --pull stream1 test-consumer-1
/nats consumer create --defaults --pull stream1 test-consumer-2
/nats consumer create --defaults --pull stream1 test-consumer-3
/nats consumer create --defaults --pull stream2 test-consumer-1
/nats consumer create --defaults --pull stream2 test-consumer-2
/nats consumer create --defaults --pull stream2 test-consumer-3

# Publish some messages
/nats publish --jetstream "stream1.testing" "this is a test"
/nats publish --jetstream "stream2.testing" "this is a test"

# Some other useful benchmarks
/nats bench pub foo --clients 10 --msgs 10000 --size 512 --no-progress
/nats bench sub foo --clients 5 --msgs 10000 --no-progress
/nats bench service serve --clients 4 testservice --no-progress
/nats bench service request --clients 4 testservice --msgs 20000 --no-progress

# Sleep after benchmarks are done to keep container running with lower CPU usage
sleep infinity
