This is the `collstats` metricset of the module mongodb.

## Overview

The `collstats` metricset collects collection-level operational and storage statistics from MongoDB. It transparently uses either the deprecated `collStats` database command (legacy) or the `$collStats` aggregation stage (MongoDB 6.2+) depending on server version and feature availability.

Notes:
- For sharded clusters, stats from multiple shards are merged. The metricset reports an aggregate view and exposes a `shardCount` summary. It does not emit a per-shard breakdown (no `shards.*`).
- Index size details (the `indexSizes.*` map) are intentionally not collected at this time.

## Features

### Operation Statistics
The `collstats` metricset uses the `top` administrative command to return usage statistics for each collection. It provides the amount of time, in microseconds, used and a count of operations for the following types:

* total
* readLock
* writeLock
* queries
* getmore
* insert
* update
* remove
* commands

It requires the following privileges, which is covered by the [`clusterMonitor` role](https://docs.mongodb.com/manual/reference/built-in-roles/#clusterMonitor):

* [`top` action](https://docs.mongodb.com/manual/reference/privilege-actions/#top) on [`cluster` resource](https://docs.mongodb.com/manual/reference/resource-document/#cluster-resource)
* [`collStats` action](https://docs.mongodb.com/manual/reference/privilege-actions/#collStats) on collection resources
* [`aggregate` action](https://docs.mongodb.com/manual/reference/privilege-actions/#aggregate) on collection resources (for MongoDB 6.2+)

On mongos routers, the `top` command is not available. In such cases, only storage statistics are populated; operation counters (total/read/write/query, etc.) may be absent.

## Configuration

Optional settings for this metricset:

- `scale` (integer, default: 1): Server-side scale factor for size values reported by `collStats`/`$collStats` (for example, set to `1024` to receive sizes in KiB). Values are not rescaled client-side.