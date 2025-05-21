---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-memcached.html
---

<!-- This file is generated! See scripts/mage/docs_collector.go -->

# Memcached module [metricbeat-module-memcached]

This is the Memcached module. These metricsets were tested with Memcached version 1.4.35.

The default metricset is `stats`.


## Compatibility [_compatibility_33]

The memcached module is tested with memcached 1.4.35.


## Example configuration [_example_configuration]

The Memcached module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: memcached
  metricsets: ["stats"]
  period: 10s
  hosts: ["localhost:11211"]
  enabled: true
```


## Metricsets [_metricsets_73]

The following metricsets are available:

* [stats](/reference/metricbeat/metricbeat-metricset-memcached-stats.md)
