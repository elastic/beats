#!/usr/bin/env bash
set -euo pipefail

# Detect Docker Compose project prefix - defaults to current directory name
if [ -z "${COMPOSE_PROJECT_NAME:-}" ]; then
    PROJECT_PREFIX=$(basename "$(pwd)")
else
    PROJECT_PREFIX="$COMPOSE_PROJECT_NAME"
fi

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

# Container names with project prefix
CONFIG1="${PROJECT_PREFIX}-config1-1"
CONFIG2="${PROJECT_PREFIX}-config2-1"
CONFIG3="${PROJECT_PREFIX}-config3-1"
SHARD1_PRIMARY="${PROJECT_PREFIX}-shard1-primary-1"
SHARD1_SECONDARY="${PROJECT_PREFIX}-shard1-secondary-1"
SHARD1_ARBITER="${PROJECT_PREFIX}-shard1-arbiter-1"
SHARD2_PRIMARY="${PROJECT_PREFIX}-shard2-primary-1"
SHARD2_SECONDARY="${PROJECT_PREFIX}-shard2-secondary-1"
SHARD2_ARBITER="${PROJECT_PREFIX}-shard2-arbiter-1"
MONGOS1="${PROJECT_PREFIX}-mongos1-1"

echo "Using container prefix: ${PROJECT_PREFIX}"
echo "Waiting for containers to be ready..."
sleep 10

# Init config RS
echo "Initializing config replica set..."
CFG_SHELL=$(shell_cmd "$CONFIG1")
docker exec "$CONFIG1" $CFG_SHELL --quiet --port 27019 --eval "rs.initiate({_id:'cfgRS',configsvr:true,members:[{_id:0,host:'${CONFIG1}:27019'},{_id:1,host:'${CONFIG2}:27019'},{_id:2,host:'${CONFIG3}:27019'}]})"

# Wait for config RS to elect primary
echo "Waiting for config RS to elect primary..."
retry docker exec "$CONFIG1" $CFG_SHELL --quiet --port 27019 --eval "rs.status().members.find(m => m.stateStr === 'PRIMARY')"

# Init shard01 (without arbiters for MongoDB 7.0 compatibility)
echo "Initializing shard01..."
S1_SHELL=$(shell_cmd "$SHARD1_PRIMARY")
docker exec "$SHARD1_PRIMARY" $S1_SHELL --quiet --port 27018 --eval "rs.initiate({_id:'shard01',members:[{_id:0,host:'${SHARD1_PRIMARY}:27018',priority:2},{_id:1,host:'${SHARD1_SECONDARY}:27018',priority:1}]})"

# Wait for shard01 primary
echo "Waiting for shard01 primary..."
retry docker exec "$SHARD1_PRIMARY" $S1_SHELL --quiet --port 27018 --eval "rs.status().members.find(m => m.stateStr === 'PRIMARY')"

# Init shard02 (without arbiters for MongoDB 7.0 compatibility)
echo "Initializing shard02..."
S2_SHELL=$(shell_cmd "$SHARD2_PRIMARY")
docker exec "$SHARD2_PRIMARY" $S2_SHELL --quiet --port 27018 --eval "rs.initiate({_id:'shard02',members:[{_id:0,host:'${SHARD2_PRIMARY}:27018',priority:2},{_id:1,host:'${SHARD2_SECONDARY}:27018',priority:1}]})"

# Wait for shard02 primary
echo "Waiting for shard02 primary..."
retry docker exec "$SHARD2_PRIMARY" $S2_SHELL --quiet --port 27018 --eval "rs.status().members.find(m => m.stateStr === 'PRIMARY')"

# Wait for mongos to be reachable
echo "Waiting for mongos to be ready..."
MONGOS_SHELL=$(shell_cmd "$MONGOS1")
sleep 10
retry docker exec "$MONGOS1" $MONGOS_SHELL --quiet --eval 'db.runCommand({ ping: 1 })'

# Add shards
echo "Adding shards to cluster..."
docker exec "$MONGOS1" $MONGOS_SHELL --quiet --eval "sh.addShard('shard01/${SHARD1_PRIMARY}:27018')"
docker exec "$MONGOS1" $MONGOS_SHELL --quiet --eval "sh.addShard('shard02/${SHARD2_PRIMARY}:27018')"

# Enable sharding for database
echo "Enabling sharding for mbtest database..."
docker exec "$MONGOS1" $MONGOS_SHELL --quiet --eval 'sh.enableSharding("mbtest")'

# Create and shard collections
echo "Creating and sharding collections..."
cat <<'JS' | docker exec -i "$MONGOS1" $MONGOS_SHELL --quiet
use mbtest

// Hash-sharded collection
db.coll_hash.drop()
db.createCollection('coll_hash')
sh.shardCollection('mbtest.coll_hash', { userId: 'hashed' })

// Seed some docs
for (let i = 0; i < 20000; i++) {
  db.coll_hash.insertOne({ _id: i, userId: i % 1000, payload: 'y'.repeat((i % 128) + 1) })
}
db.coll_hash.createIndex({ userId: 1 })

// Range-sharded collection
db.coll_range.drop()
db.createCollection('coll_range')
sh.shardCollection('mbtest.coll_range', { userId: 1 })
sh.splitAt('mbtest.coll_range', { userId: 0 })
sh.moveChunk('mbtest.coll_range', { userId: -1 }, 'shard01')
sh.moveChunk('mbtest.coll_range', { userId: 1 }, 'shard02')

// Seed ranged docs
for (let i = -10000; i < 0; i++) {
  db.coll_range.insertOne({ _id: i, userId: i, payload: 'z'.repeat(((-i) % 64) + 1) })
}
for (let i = 1; i <= 10000; i++) {
  db.coll_range.insertOne({ _id: i, userId: i, payload: 'z'.repeat((i % 64) + 1) })
}
db.coll_range.createIndex({ userId: 1 })

print('Sharded cluster initialized and seeded')
JS

echo "Done initializing sharded cluster."
echo "You can now connect to mongos at localhost:27017"