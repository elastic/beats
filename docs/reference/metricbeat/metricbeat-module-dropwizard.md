---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-dropwizard.html
---

# Dropwizard module [metricbeat-module-dropwizard]

This is the [Dropwizard](http://dropwizard.io) module. The default metricset is `collector`.


## Compatibility [_compatibility_17]

The Dropwizard module is tested with dropwizard metrics 3.2.6, 4.0.0 and 4.1.2.


## Example configuration [_example_configuration_19]

The Dropwizard module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: dropwizard
  metricsets: ["collector"]
  period: 10s
  hosts: ["localhost:8080"]
  metrics_path: /metrics/metrics
  namespace: example
  enabled: true
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_25]

The following metricsets are available:

* [collector](/reference/metricbeat/metricbeat-metricset-dropwizard-collector.md)


