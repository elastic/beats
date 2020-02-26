#!/bin/bash

/nats-streaming-server -m 8222 &
sleep 2
/stan-bench -np 0 -ns 100 -qgroup T -n 100000000 -ms 1024 foo &
/stan-bench -np 10 -ns 10 -n 1000000000 -ms 1024 bar &

# Make sure the container keeps running
tail -f /dev/null

