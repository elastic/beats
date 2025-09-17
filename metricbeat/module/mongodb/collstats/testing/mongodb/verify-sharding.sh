#!/usr/bin/env bash
set -euo pipefail

# Detect Docker Compose project prefix
PROJECT_PREFIX="${COMPOSE_PROJECT_NAME:-mongodb}"
MONGOS="${PROJECT_PREFIX}-mongos-1"

shell_cmd() {
  local cn=$1; shift
  if docker exec "$cn" mongosh --quiet "$@" >/dev/null 2>&1; then
    echo "mongosh"
    return
  fi
  echo "mongo"
}

ROUTER_SHELL=$(shell_cmd "$MONGOS")

echo "Using container: ${MONGOS}"

# Basic cluster state
echo "--- sh.status() summary ---"
docker exec "$MONGOS" $ROUTER_SHELL --quiet --eval 'sh.status({verbose:0})'

# Chunk distribution
cat <<'JS' | docker exec -i "$MONGOS" $ROUTER_SHELL --quiet
use config
print('\n--- Chunks for mbtest.coll_range ---')
printjson(db.chunks.find({ ns: 'mbtest.coll_range' }).toArray())
print('\n--- Chunks for mbtest.coll_hash (first 5) ---')
printjson(db.chunks.find({ ns: 'mbtest.coll_hash' }).limit(5).toArray())
JS

# Quick collStats checks via legacy command and aggregation
cat <<'JS' | docker exec -i "$MONGOS" $ROUTER_SHELL --quiet
use mbtest

function runCollStatsLegacy(coll) {
  const res = db.runCommand({ collStats: coll })
  print('\nLegacy collStats for', coll)
  printjson({count: res.count, size: res.size, storageSize: res.storageSize, totalIndexSize: res.totalIndexSize, shardCount: res.shards ? Object.keys(res.shards).length : 0})

  if (res.shards) {
    // Sum shard fields
    let sum = { count: 0, size: 0, storageSize: 0, totalIndexSize: 0 }
    let weightedAvgSizeNum = 0
    let weightedAvgSizeDen = 0
    let mergedIndexSizes = {}
    Object.keys(res.shards).forEach(sh => {
      const s = res.shards[sh]
      sum.count += (s.count || 0)
      sum.size += (s.size || 0)
      sum.storageSize += (s.storageSize || 0)
      sum.totalIndexSize += (s.totalIndexSize || 0)
      if (s.avgObjSize && s.count) {
        weightedAvgSizeNum += s.avgObjSize * s.count
        weightedAvgSizeDen += s.count
      }
      if (s.indexSizes) {
        Object.keys(s.indexSizes).forEach(ix => {
          mergedIndexSizes[ix] = (mergedIndexSizes[ix] || 0) + (s.indexSizes[ix] || 0)
        })
      }
    })

    // Assertions
    function eq(a, b) { return Number(a) === Number(b) }
    const okCount = eq(sum.count, res.count)
    const okSize = eq(sum.size, res.size)
    const okStorage = eq(sum.storageSize, res.storageSize)
    const okIdx = eq(sum.totalIndexSize, res.totalIndexSize)
    const mergedAvg = weightedAvgSizeDen ? (weightedAvgSizeNum / weightedAvgSizeDen) : 0
    const okAvg = Math.abs((res.avgObjSize || 0) - mergedAvg) < 1e-9

    print('Legacy shard merge checks:')
    print('  count sum == top-level:', okCount)
    print('  size sum == top-level:', okSize)
    print('  storageSize sum == top-level:', okStorage)
    print('  totalIndexSize sum == top-level:', okIdx)
    print('  avgObjSize weighted average matches:', okAvg)

    if (res.indexSizes) {
      let okAllIx = true
      Object.keys(mergedIndexSizes).forEach(ix => {
        const ok = eq(mergedIndexSizes[ix], res.indexSizes[ix] || 0)
        if (!ok) okAllIx = false
      })
      print('  indexSizes per-index sums match top-level:', okAllIx)
    }
  }
}

function runCollStatsAgg(coll) {
  const cursor = db.getCollection(coll).aggregate([{ $collStats: { storageStats: {}, count: {} } }])
  const docs = cursor.toArray()
  print('\nAgg $collStats for', coll)
  if (docs.length === 1) {
    const d = docs[0]
    const s = d.storageStats || {}
    printjson({count: (d.count||{}).count || s.count, size: s.size, storageSize: s.storageSize, totalIndexSize: s.totalIndexSize})
  } else {
    printjson({ shards: docs.length })
    // Merge shard docs manually and compare to mongos legacy collStats
    let sum = { count: 0, size: 0, storageSize: 0, totalIndexSize: 0 }
    let weightedAvgSizeNum = 0
    let weightedAvgSizeDen = 0
    let mergedIndexSizes = {}
    docs.forEach(d => {
      const s = d.storageStats || {}
      const count = ((d.count||{}).count) || s.count || 0
      sum.count += (count || 0)
      sum.size += (s.size || 0)
      sum.storageSize += (s.storageSize || 0)
      sum.totalIndexSize += (s.totalIndexSize || 0)
      if (s.avgObjSize && count) {
        weightedAvgSizeNum += s.avgObjSize * count
        weightedAvgSizeDen += count
      }
      if (s.indexSizes) {
        Object.keys(s.indexSizes).forEach(ix => {
          mergedIndexSizes[ix] = (mergedIndexSizes[ix] || 0) + (s.indexSizes[ix] || 0)
        })
      }
    })
    const mergedAvg = weightedAvgSizeDen ? (weightedAvgSizeNum / weightedAvgSizeDen) : 0
    printjson({ mergedFromAgg: sum, weightedAvgObjSize: mergedAvg, mergedIndexSizes })
  }
}

runCollStatsLegacy('coll_range')
runCollStatsLegacy('coll_hash')
runCollStatsAgg('coll_range')
runCollStatsAgg('coll_hash')
JS
