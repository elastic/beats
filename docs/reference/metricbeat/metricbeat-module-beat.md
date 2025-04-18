---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-beat.html
---

# Beat module [metricbeat-module-beat]

The `beat` module collects metrics about any Beat or other software based on libbeat.


## Compatibility [_compatibility_8]

The `beat` module works with {{beats}} 7.3.0 and later.


## Usage for {{stack}} Monitoring [_usage_for_stack_monitoring]

The `beat` module can be used to collect metrics shown in our {{stack-monitor-app}} UI in {{kib}}. To enable this usage, set `xpack.enabled: true` and remove any `metricsets` from the module’s configuration. Alternatively, run `metricbeat modules disable beat` and `metricbeat modules enable beat-xpack`.

::::{note}
When this module is used for {{stack}} Monitoring, it sends metrics to the monitoring index instead of the default index typically used by {{metricbeat}}. For more details about the monitoring index, see [Configuring indices for monitoring](docs-content://deploy-manage/monitor/monitoring-data/configuring-data-streamsindices-for-monitoring.md).
::::



## Example configuration [_example_configuration_8]

The Beat module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: beat
  metricsets:
    - stats
    - state
  period: 10s
  hosts: ["http://localhost:5066"]
  #ssl.certificate_authorities: ["/etc/pki/root/ca.pem"]

  # Set to true to send data collected by module to X-Pack
  # Monitoring instead of metricbeat-* indices.
  #xpack.enabled: false
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_12]

The following metricsets are available:

* [state](/reference/metricbeat/metricbeat-metricset-beat-state.md)
* [stats](/reference/metricbeat/metricbeat-metricset-beat-stats.md)



