---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/_metricsets_70.html
  # That link will 404 until 8.18 is current
  # (see https://www.elastic.co/guide/en/beats/metricbeat/8.18/_metricsets_70.html)
---

# Metricsets [_metricsets_70]

Currently, there is only `server` metricset in `statsd` module.


### `server` [_server]

The metricset collects metric data sent using UDP and publishes them under the `statsd` prefix.


## Example configuration [_example_configuration_61]

The Statsd module supports the standard configuration options that are described in [Modules](configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: statsd
  host: "localhost"
  port: "8125"
  enabled: false
  #ttl: "30s"
```


## Metricsets [_metricsets_71]

The following metricsets are available:

* [server](metricbeat-metricset-statsd-server.md)

