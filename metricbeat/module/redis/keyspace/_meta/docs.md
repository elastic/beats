The Redis `keyspace` metricset collects information about the Redis keyspaces. For each keyspace, an event is sent to Elasticsearch. The keyspace information is fetched from the [`INFO`](http://redis.io/commands/INFO) command.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
