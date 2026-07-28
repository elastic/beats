#!/usr/bin/env bash
set -euo pipefail

# Detect Docker Compose project prefix
if [ -z "${COMPOSE_PROJECT_NAME:-}" ]; then
    PROJECT_PREFIX=$(basename "$(pwd)")
else
    PROJECT_PREFIX="$COMPOSE_PROJECT_NAME"
fi

MONGOS="${PROJECT_PREFIX}-mongos1-1"

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

ROUTER_SHELL=$(shell_cmd "$MONGOS")

echo "Using container: ${MONGOS}"
echo "Verifying sharding setup..."

# Show basic cluster status
echo "Cluster status:"
docker exec "$MONGOS" $ROUTER_SHELL --quiet --eval 'sh.status({verbose:0})'

# Check chunk distribution
echo "Checking chunk distribution..."
cat <<'JS' | docker exec -i "$MONGOS" $ROUTER_SHELL --quiet
use config
print('--- Chunks for mbtest.coll_range ---')
db.chunks.find({ ns: 'mbtest.coll_range' }).forEach(function(c) {
    print('Chunk on ' + c.shard + ': ' + JSON.stringify(c.min) + ' to ' + JSON.stringify(c.max))
})
print('--- Chunks for mbtest.coll_hash (sample) ---')
db.chunks.find({ ns: 'mbtest.coll_hash' }).limit(5).forEach(function(c) {
    print('Chunk on ' + c.shard + ': ' + JSON.stringify(c.min) + ' to ' + JSON.stringify(c.max))
})
JS

# Check collection stats
echo "Collection statistics:"
cat <<'JS' | docker exec -i "$MONGOS" $ROUTER_SHELL --quiet
use mbtest

// Check coll_hash
const hashStats = db.runCommand({ collStats: 'coll_hash' })
print('coll_hash:')
print('  Count: ' + hashStats.count)
print('  Size: ' + Math.round(hashStats.size / 1024) + ' KB')
print('  Shards: ' + (hashStats.shards ? Object.keys(hashStats.shards).length : 0))

// Check coll_range
const rangeStats = db.runCommand({ collStats: 'coll_range' })
print('coll_range:')
print('  Count: ' + rangeStats.count)
print('  Size: ' + Math.round(rangeStats.size / 1024) + ' KB')
print('  Shards: ' + (rangeStats.shards ? Object.keys(rangeStats.shards).length : 0))

// Verify sharding is working
if (hashStats.shards && Object.keys(hashStats.shards).length > 1) {
    print('✓ coll_hash is properly sharded across multiple shards')
} else {
    print('✗ coll_hash is not properly distributed')
}

if (rangeStats.shards && Object.keys(rangeStats.shards).length > 1) {
    print('✓ coll_range is properly sharded across multiple shards')
} else {
    print('✗ coll_range is not properly distributed')
}
JS

echo "Verification complete."