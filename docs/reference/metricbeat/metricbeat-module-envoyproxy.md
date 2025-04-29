---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-envoyproxy.html
---

# Envoyproxy module [metricbeat-module-envoyproxy]

This is the envoyproxy module.

The default metricset is `server`.


## Compatibility [_compatibility_20]

The envoyproxy module is tested with Envoy 1.7.0 and 1.12.0.


## Example configuration [_example_configuration_22]

The Envoyproxy module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: envoyproxy
  metricsets: ["server"]
  period: 10s
  hosts: ["localhost:9901"]
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_28]

The following metricsets are available:

* [server](/reference/metricbeat/metricbeat-metricset-envoyproxy-server.md)


