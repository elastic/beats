This is the `collstats` metricset of the module mongodb.

It primarily uses the `top` administrative command to return usage statistics for each collection. On MongoDB 6.2+ it will attempt to enrich collection statistics via the `$collStats` aggregation stage (with server–side scaling and a fallback to the legacy `collStats` command if aggregation is not available). It provides the amount of time, in microseconds, used and a count of operations for the following types:

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

### Additional collection statistics (MongoDB 6.2+)

When connected to MongoDB 6.2 or later, Metricbeat will:

* Detect server version during the first fetch.
* Try `$collStats` aggregation with a dynamic stage including `storageStats` (with optional scale), `count` (fast metadata count)
* Fall back to the legacy `collStats` command if the aggregation stage fails (e.g. older server, view, permissions, or feature restrictions such as Queryable Encryption redaction).

### Configuration options

The following optional settings control the output (default scale `1`):

```
	# Scale factor for size values reported inside stats.* (1 = bytes, 1024 = KiB, etc.).
	# Only applied server‑side; Metricbeat does not rescale client‑side.
	#scale: 1
```

Notes:
* Sharded collections returned by `$collStats` produce one doc per shard; the metricset merges these and adds `stats.shards[]` plus `stats.shardCount`.
* The server’s reported sizes (e.g. `storageSize`, `totalIndexSize`) already reflect the `scale` value supplied; no additional scaling is performed.

Backward compatibility: For MongoDB < 6.2 the behavior remains identical to previous versions, using only the legacy command path.
