This is the `dbstats` metricset of the MongoDB module.

It requires the following privileges, which is covered by the [`clusterMonitor` role](https://docs.mongodb.com/manual/reference/built-in-roles/#clusterMonitor):

* [`listDatabases` action](https://docs.mongodb.com/manual/reference/privilege-actions/#listDatabases) on [`cluster` resource](https://docs.mongodb.com/manual/reference/resource-document/#cluster-resource)
* for each of the databases, also need [`dbStats` action](https://docs.mongodb.com/manual/reference/privilege-actions/#dbStats) on the [`database` resource](https://docs.mongodb.com/manual/reference/resource-document/#database-and-or-collection-resource)
