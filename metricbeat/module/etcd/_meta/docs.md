This module targets Etcd V2 and V3.

When using V2, metrics are collected using [Etcd v2 API](https://coreos.com/etcd/docs/latest/v2/api.md). When using V3, metrics are retrieved from the `/metrics` endpoint as intended for [Etcd v3](https://coreos.com/etcd/docs/latest/metrics.md)

When using V3, metricsest are bundled into `metrics` When using V2, metricsets available are `leader`, `self` and `store`.


## Compatibility [_compatibility_21]

The etcd module is tested with etcd 3.2 and 3.3.
