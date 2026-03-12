This is the `field_stats` metricset of the Elasticsearch module.

It collects per-field, per-shard usage statistics from the Elasticsearch
Field Usage Stats API (`GET /{index}/_field_usage_stats`). This is useful for
identifying which fields in your indices are actively being used in queries,
aggregations, and other operations.

Because this metricset can produce a large volume of events (one per field per
shard per index), it is recommended to configure a longer collection period,
for example `period: 10m`.
