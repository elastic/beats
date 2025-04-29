---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-etcd.html
---

# Etcd module [metricbeat-module-etcd]

This module targets Etcd V2 and V3.

When using V2, metrics are collected using [Etcd v2 API](https://coreos.com/etcd/docs/latest/v2/api.md). When using V3, metrics are retrieved from the `/metrics` endpoint as intended for [Etcd v3](https://coreos.com/etcd/docs/latest/metrics.md)

When using V3, metricsest are bundled into `metrics` When using V2, metricsets available are `leader`, `self` and `store`.


## Compatibility [_compatibility_21]

The etcd module is tested with etcd 3.2 and 3.3.


## Example configuration [_example_configuration_23]

The Etcd module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: etcd
  metricsets: ["leader", "self", "store"]
  period: 10s
  hosts: ["localhost:2379"]
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_29]

The following metricsets are available:

* [leader](/reference/metricbeat/metricbeat-metricset-etcd-leader.md)
* [metrics](/reference/metricbeat/metricbeat-metricset-etcd-metrics.md)
* [self](/reference/metricbeat/metricbeat-metricset-etcd-self.md)
* [store](/reference/metricbeat/metricbeat-metricset-etcd-store.md)





