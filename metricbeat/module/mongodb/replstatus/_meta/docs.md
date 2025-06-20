This is the `replstatus` metricset of the module mongodb.

It requires the following privileges, which is covered by the [`clusterMonitor` role](https://docs.mongodb.com/manual/reference/built-in-roles/#clusterMonitor):

* [`find`/`listCollections` action](https://docs.mongodb.com/manual/reference/privilege-actions/#find) on [the `local` database resource](https://docs.mongodb.com/manual/reference/local-database/)
* [`collStats` action](https://docs.mongodb.com/manual/reference/privilege-actions/#collStats) on [the `local.oplog.rs` collection resource](https://docs.mongodb.com/manual/reference/local-database/#local.oplog.rs)
* [`replSetGetStatus` action](https://docs.mongodb.com/manual/reference/privilege-actions/#replSetGetStatus) on [`cluster` resource](https://docs.mongodb.com/manual/reference/resource-document/#cluster-resource)
