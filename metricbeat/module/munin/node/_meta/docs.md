This is the node metricset of the module munin.


## Features and configuration [_features_and_configuration_3]

The node metricset of the munin module collects metrics from a munin node agent and sends them as events to Elastic.

```yaml

```

Metrics exposed by a single munin node will be sent in an event per plugin.

For example with the previous configuration two events are sent like the following ones.

```json
---
"munin": {
  "plugin": {
    "name": "swap"
  },
  "metrics": {
    "swap_in": 198609,
    "swap_out": 612629
  }
}
```

"munin": { "plugin": { "name": "cpu" } "metrics": { "softirq": 680, "guest": 0, "user": 158212, "iowait": 71095, "irq": 1, "system": 35906, "idle": 1185709, "steal": 0, "nice": 1633 } } ---

In principle this module can be used to collect metrics from any agent that implements the munin node protocol ([http://guide.munin-monitoring.org/en/latest/master/network-protocol.html](http://guide.munin-monitoring.org/en/latest/master/network-protocol.html)).


## Limitations [_limitations_2]

Currently this module only collects metrics using the basic protocol. It doesn’t support capabilities or automatic dashboards generation based on munin configuration.


## Exposed fields, dashboards, indexes, etc. [_exposed_fields_dashboards_indexes_etc_3]

Munin supports a great variety of plugins each of them can be used to obtain different sets of metrics. Metricbeat cannot know the metrics exposed beforehand, so no field description or dashboard is generated automatically.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
