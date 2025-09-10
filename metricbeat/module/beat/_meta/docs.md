The `beat` module collects metrics about any Beat or other software based on libbeat.


## Compatibility [_compatibility_8]

The `beat` module works with {{beats}} 7.3.0 and later.


## Usage for {{stack}} Monitoring [_usage_for_stack_monitoring]

The `beat` module can be used to collect metrics shown in our {{stack-monitor-app}} UI in {{kib}}. To enable this usage, set `xpack.enabled: true` and remove any `metricsets` from the module’s configuration. Alternatively, run `metricbeat modules disable beat` and `metricbeat modules enable beat-xpack`.

::::{note}
When this module is used for {{stack}} Monitoring, it sends metrics to the monitoring index instead of the default index typically used by {{metricbeat}}. For more details about the monitoring index, see [Configuring indices for monitoring](docs-content://deploy-manage/monitor/monitoring-data/configuring-data-streamsindices-for-monitoring.md).
::::
