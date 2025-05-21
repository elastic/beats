This is the `collstats` metricset of the module mongodb.

It is using the `top` administrative command to return usage statistics for each collection. It provides the amount of time, in microseconds, used and a count of operations for the following types:

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

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
