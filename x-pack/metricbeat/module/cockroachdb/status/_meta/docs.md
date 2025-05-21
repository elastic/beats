The CockroachDB `status` metricset collects metrics exposed by the [Prometheus endpoint](https://www.cockroachlabs.com/docs/v2.1/monitoring-and-alerting.html#prometheus-endpoint) of CockroachDB.

::::{warning}
This metricset collects a large number of metrics, what can significantly impact disk usage. Processors can be used to drop unused metrics before they are stored. For example the following configuration will drop all histogram buckets:
::::


```yaml
- module: cockroachdb
  metricsets: ['status']
  hosts: ['${data.host}:8080']
  processors:
    - drop_event.when.has_fields: ['prometheus.labels.le']
```

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
