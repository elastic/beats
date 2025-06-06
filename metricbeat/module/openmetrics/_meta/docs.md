::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This module periodically fetches metrics from endpoints following [Openmetrics](https://openmetrics.io/) format.


## Filtering metrics [_filtering_metrics]

In order to filter out/in metrics one can make use of `metrics_filters.include` `metrics_filters.exclude` settings:

```yaml
- module: openmetrics
  metricsets: ['collector']
  period: 10s
  hosts: ["localhost:9090"]
  metrics_path: /metrics
  metrics_filters:
    include: ["node_filesystem_*"]
    exclude: ["node_filesystem_device_*", "^node_filesystem_readonly$"]
```

The configuration above will include only metrics that match `node_filesystem_*` pattern and do not match `node_filesystem_device_*` and are not `node_filesystem_readonly` metric.
