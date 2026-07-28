#!/usr/bin/env bash
set -euo pipefail

# Detect Docker Compose project prefix - defaults to current directory name
if [ -z "${COMPOSE_PROJECT_NAME:-}" ]; then
    PROJECT_PREFIX=$(basename "$(pwd)")
else
    PROJECT_PREFIX="$COMPOSE_PROJECT_NAME"
fi
RS_NAME="${RS_NAME:-rs0}"

retry() {
  local n=0
  until "$@"; do
    n=$((n+1))
    if [ $n -ge 30 ]; then
      return 1
    fi
    sleep 3
  done
}

shell_cmd() {
  local cn=$1; shift
  # Try mongosh first (MongoDB 5.0+)
  if docker exec "$cn" which mongosh >/dev/null 2>&1; then
    echo "mongosh"
    return
  fi
  # Fall back to mongo for older versions
  echo "mongo"
}

# Container names
PRIMARY="${PROJECT_PREFIX}-mongo-primary-1"
SECONDARY1="${PROJECT_PREFIX}-mongo-secondary1-1"
SECONDARY2="${PROJECT_PREFIX}-mongo-secondary2-1"
ARBITER="${PROJECT_PREFIX}-mongo-arbiter-1"

echo "Starting replica set initialization"
echo "Project prefix: $PROJECT_PREFIX"
echo "Replica set name: $RS_NAME"

# Wait for containers
echo "Waiting for containers to be ready..."
sleep 5

# Detect shell command
SHELL_CMD=$(shell_cmd "$PRIMARY")
echo "Using MongoDB shell: $SHELL_CMD"

# Wait for MongoDB
echo "Waiting for MongoDB to be ready..."
retry docker exec "$PRIMARY" $SHELL_CMD --quiet --eval "db.adminCommand('ping')"

# Ensure all members' mongod are up before initiating (avoids quorum failures)
echo "Checking members readiness..."
retry docker exec "$SECONDARY1" $SHELL_CMD --quiet --eval "db.adminCommand('ping')"
retry docker exec "$SECONDARY2" $SHELL_CMD --quiet --eval "db.adminCommand('ping')"
retry docker exec "$ARBITER"    $SHELL_CMD --quiet --eval "db.adminCommand('ping')"

# Initialize replica set using service names for stable DNS within the compose network
echo "Initializing replica set '$RS_NAME'..."
cat <<JS | docker exec -i "$PRIMARY" $SHELL_CMD --quiet
rs.initiate({
    _id: "$RS_NAME",
    members: [
        { _id: 0, host: "mongo-primary:27017",   priority: 2 },
        { _id: 1, host: "mongo-secondary1:27017", priority: 1 },
        { _id: 2, host: "mongo-secondary2:27017", priority: 1 },
        { _id: 3, host: "mongo-arbiter:27017",    priority: 0, arbiterOnly: true }
    ]
})
JS

# Wait for primary election (ensure this node is writable primary)
echo "Waiting for primary election to complete..."
retry docker exec "$PRIMARY" $SHELL_CMD --quiet --eval "var h = (typeof db.hello === 'function') ? db.hello() : db.isMaster(); if (!(h.isWritablePrimary || h.ismaster === true)) { quit(1) }"

# Seed test data
echo "Seeding test data into mbtest database..."
cat <<'JS' | docker exec -i "$PRIMARY" $SHELL_CMD --quiet
use mbtest

// Drop existing collections
db.test_collection.drop()
db.test_collection_indexed.drop()
db.test_collection_capped.drop()

// Create regular collection
db.createCollection("test_collection")
for (let i = 1; i <= 10000; i++) {
    db.test_collection.insert({
        _id: i,
        userId: Math.floor(Math.random() * 1000),
        timestamp: new Date(),
        data: 'x'.repeat((i % 128) + 1),
        nested: {
            field1: 'value' + i,
            field2: Math.random() * 1000
        }
    })
}

// Create indexed collection
db.createCollection("test_collection_indexed")
db.test_collection_indexed.createIndex({ userId: 1 })
db.test_collection_indexed.createIndex({ timestamp: -1 })

for (let i = 1; i <= 5000; i++) {
    db.test_collection_indexed.insert({
        _id: i,
        userId: i % 500,
        timestamp: new Date(Date.now() - Math.random() * 86400000 * 30),
        category: ['A', 'B', 'C', 'D'][i % 4],
        score: Math.random() * 100
    })
}

// Create capped collection
db.createCollection("test_collection_capped", {
    capped: true,
    size: 10485760,  // 10MB
    max: 5000
})

for (let i = 1; i <= 5000; i++) {
    db.test_collection_capped.insert({
        event: 'event_' + i,
        timestamp: new Date(),
        data: Math.random().toString(36).substring(7)
    })
}

print("Test data seeded successfully")
JS

echo "Replica set initialization completed successfully!"
echo "You can connect to the primary at: localhost:27017"
echo "Secondary nodes are available at: localhost:27018 and localhost:27019"