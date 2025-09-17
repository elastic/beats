#!/usr/bin/env bash
set -euo pipefail

# Detect Docker Compose project prefix
PROJECT_PREFIX="${COMPOSE_PROJECT_NAME:-mongodb}"

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
SHARD1="${PROJECT_PREFIX}-shard1-1"
SHARD2="${PROJECT_PREFIX}-shard2-1"
MONGOS="${PROJECT_PREFIX}-mongos-1"

echo "Using container prefix: ${PROJECT_PREFIX}"
echo "Waiting for containers to be ready..."
sleep 5

# Init cfgRS
echo "Initializing config replica set..."
CFG_SHELL=$(shell_cmd "$CONFIG1")
docker exec "$CONFIG1" $CFG_SHELL --quiet --port 27019 --eval "rs.initiate({_id:'cfgRS',configsvr:true,members:[{_id:0,host:'${CONFIG1}:27019'},{_id:1,host:'${CONFIG2}:27019'},{_id:2,host:'${CONFIG3}:27019'}]})"

# Wait for config RS to elect primary
echo "Waiting for config RS to elect primary..."
retry docker exec "$CONFIG1" $CFG_SHELL --quiet --port 27019 --eval "rs.status().members.find(m => m.stateStr === 'PRIMARY')"

# Init shard01
echo "Initializing shard01..."
S1_SHELL=$(shell_cmd "$SHARD1")
docker exec "$SHARD1" $S1_SHELL --quiet --port 27018 --eval "rs.initiate({_id:'shard01',members:[{_id:0,host:'${SHARD1}:27018'}]})"

# Init shard02
echo "Initializing shard02..."
S2_SHELL=$(shell_cmd "$SHARD2")
docker exec "$SHARD2" $S2_SHELL --quiet --port 27018 --eval "rs.initiate({_id:'shard02',members:[{_id:0,host:'${SHARD2}:27018'}]})"

# Wait for mongos to be reachable
echo "Waiting for mongos to be ready..."
MONGOS_SHELL=$(shell_cmd "$MONGOS")
echo "Waiting for mongos process to start..."
sleep 10
retry docker exec "$MONGOS" $MONGOS_SHELL --quiet --eval 'db.runCommand({ ping: 1 })'

# Add shards
echo "Adding shards to cluster..."
ROUTER_SHELL=$(shell_cmd "$MONGOS")
docker exec "$MONGOS" $ROUTER_SHELL --quiet --eval "sh.addShard('shard01/${SHARD1}:27018')"
docker exec "$MONGOS" $ROUTER_SHELL --quiet --eval "sh.addShard('shard02/${SHARD2}:27018')"

# Enable sharding for database
echo "Enabling sharding for mbtest database..."
docker exec "$MONGOS" $ROUTER_SHELL --quiet --eval 'sh.enableSharding("mbtest")'

# Create and shard collections
echo "Creating and sharding collections..."
cat <<'JS' | docker exec -i "$MONGOS" $ROUTER_SHELL --quiet
use mbtest

db.coll_hash.drop()
db.createCollection('coll_hash')
sh.shardCollection('mbtest.coll_hash', { userId: 'hashed' })

// Seed some docs
for (let i = 0; i < 20000; i++) {
  db.coll_hash.insertOne({ _id: i, userId: i % 1000, payload: 'y'.repeat((i % 128) + 1) })
}
db.coll_hash.createIndex({ userId: 1 })

// 2) Ranged shard key, pre-split and move chunks
DB = db
DB.coll_range.drop()
db.createCollection('coll_range')
sh.shardCollection('mbtest.coll_range', { userId: 1 })
sh.splitAt('mbtest.coll_range', { userId: 0 })
sh.moveChunk('mbtest.coll_range', { userId: -1 }, 'shard01')
sh.moveChunk('mbtest.coll_range', { userId: 1 }, 'shard02')

// Seed ranged docs skewed to each side
for (let i = -10000; i < 0; i++) {
  DB.coll_range.insertOne({ _id: i, userId: i, payload: 'z'.repeat(((-i) % 64) + 1) })
}
for (let i = 1; i <= 10000; i++) {
  DB.coll_range.insertOne({ _id: i, userId: i, payload: 'z'.repeat((i % 64) + 1) })
}
DB.coll_range.createIndex({ userId: 1 })

print('Sharded cluster initialized and seeded')
JS

echo "Done initializing sharded cluster."
echo "You can now connect to mongos at localhost:27017"