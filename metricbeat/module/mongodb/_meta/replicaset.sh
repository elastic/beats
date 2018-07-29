#!/bin/bash
echo "Waiting for startup.."
until curl http://mongodb:27017/serverStatus\?text\=1 2>&1 | grep uptime | head -1; do
  printf '.'
  sleep 1
done

echo curl http://mongodb:27017/serverStatus\?text\=1 | grep uptime | head -1
echo "MongoDB Started.."

mongo --host mongodb:27017 --eval "if(rs.status().code == 94) rs.initiate();"

tail -f /dev/null
