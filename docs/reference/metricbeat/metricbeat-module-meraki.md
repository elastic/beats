---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-meraki.html
---

# Cisco Meraki module [metricbeat-module-meraki]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the meraki module.


## Example configuration [_example_configuration_42]

The Cisco Meraki module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: meraki
  metricsets: ["device_health"]
  enabled: true
  period: 300s
  apiKey: "Meraki dashboard API key"
  organizations: ["Meraki organization ID"]
```


## Metricsets [_metricsets_48]

The following metricsets are available:

* [device_health](/reference/metricbeat/metricbeat-metricset-meraki-device_health.md)


