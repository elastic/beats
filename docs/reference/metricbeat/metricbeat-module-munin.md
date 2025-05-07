---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-munin.html
---

# Munin module [metricbeat-module-munin]

This is the munin module.

The default metricset is `node`.


## Compatibility [_compatibility_36]

Munin module should be compatible with any implementation of the munin network protocol ([http://guide.munin-monitoring.org/en/latest/master/network-protocol.html](http://guide.munin-monitoring.org/en/latest/master/network-protocol.html)), it is tested with munin node 2.0.


## Example configuration [_example_configuration_45]

The Munin module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: munin
  metricsets: ["node"]
  enabled: true
  period: 10s
  hosts: ["localhost:4949"]

  # List of plugins to collect metrics from, by default it collects from
  # all the available ones.
  #munin.plugins: []

  # If set to true, it sanitizes fields names in concordance with munin
  # implementation (all characters that are not alphanumeric, or underscore
  # are replaced by underscores).
  #munin.sanitize: false
```


## Metricsets [_metricsets_52]

The following metricsets are available:

* [node](/reference/metricbeat/metricbeat-metricset-munin-node.md)


