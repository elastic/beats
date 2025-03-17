#!/bin/sh

# Core NATS publish and subscribe:
/nats bench benchsubject --pub 1 --sub 10 --no-progress

# Request reply with queue group:
/nats bench benchsubject --sub 1 --reply --no-progress
/nats bench benchsubject --pub 10 --request --no-progress

# JetStream publish:
/nats bench benchsubject --js --purge --pub 1 --no-progress

# JetStream ordered ephemeral consumers:
/nats bench benchsubject --js --sub 10 --no-progress

# JetStream durable pull and push consumers:
/nats bench benchsubject --js --sub 5 --pull --no-progress
/nats bench benchsubject --js --sub 5 --push --no-progress

# JetStream KV put and get:
/nats bench benchsubject --kv --pub 1 --no-progress
/nats bench benchsubject --kv --sub 10 --no-progress

# Sleep after benchmarks are done to keep container running with lower CPU usage
sleep 1000000
