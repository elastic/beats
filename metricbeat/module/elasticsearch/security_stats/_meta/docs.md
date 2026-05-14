This is the `security_stats` metricset of the Elasticsearch module. It queries the Security Stats API endpoint (`GET /_security/stats`, available since Elasticsearch 9.2) to collect per-node security counters. The endpoint exposes Document Level Security (DLS) cache statistics, which are useful for spotting cache thrash, oversized working sets, and unhealthy hit/miss ratios across a fleet.

Each emitted event is enriched with `node.{name,roles,version}` (alongside `node.id`) via a single side-channel `/_nodes` call per scrape, so consumers can slice by node, role, or stack version without joining across data streams.

The `/_security/stats` endpoint is only served when the Elasticsearch security feature is enabled (`xpack.security.enabled: true`). The metricset checks `GET /_xpack` on each scrape. When security is disabled, it emits a throttled debug log, but no events.

Authorization follows the same model as `/_cluster/stats` and `/_nodes/stats`: the caller needs the `monitor` cluster privilege.
