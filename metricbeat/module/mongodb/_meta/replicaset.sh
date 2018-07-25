#!/bin/bash
echo "Waiting for startup.."
until curl http://mongodb:27017/serverStatus\?text\=1 2>&1 | grep uptime | head -1; do
  printf '.'
  sleep 1
done

echo curl http://mongodb:27017/serverStatus\?text\=1 | grep uptime | head -1
echo "Started.."

sleep 10

mongo --host mongodb:27017 <<EOF
   var cfg = {
        "_id": "rs",
        "version": 1,
        "members": [
            {
                "_id": 0,
                "host": "mongodb:27017",
                "priority": 2
            },
            {
                "_id": 1,
                "host": "mongodb_secondary1:27017",
                "priority": 0
            },
            {
                "_id": 2,
                "host": "mongodb_secondary1:27017",
                "priority": 0
            }
        ]
    };
    rs.initiate(cfg, { force: true });
    rs.reconfig(cfg, { force: true });
    db.getMongo().setReadPref('nearest');
EOF

tail -f /dev/null