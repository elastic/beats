This is the remote_write metricset of the module prometheus. This metricset can receive metrics from a Prometheus server that has configured [remote_write](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write) setting accordingly, for instance:

```yaml
remote_write:
  - url: "http://localhost:9201/write"
```

::::{tip}
In order to assure the health of the whole queue, the following configuration [parameters](https://prometheus.io/docs/practices/remote_write/#parameters) should be considered:
::::


* `max_shards`: Sets the maximum number of parallelism with which Prometheus will try to send samples to Metricbeat. It is recommended that this setting should be equal to the number of cores of the machine where Metricbeat runs. Metricbeat can handle connections in parallel and hence setting `max_shards` to the number of parallelism that Metricbeat can actually achieve is the optimal queue configuration.
* `max_samples_per_send`: Sets the number of samples to batch together for each send. Recommended values are between 100 (default) and 1000. Having a bigger batch can lead to improved throughput and in more efficient storage since Metricbeat groups metrics with the same labels into same event documents. However this will increase the memory usage of Metricbeat.
* `capacity`: It is recommended to set capacity to 3-5 times `max_samples_per_send`. Capacity sets the number of samples that are queued in memory per shard, and hence capacity should be high enough so as to be able to cover `max_samples_per_send`.
* `write_relabel_configs`: It is a relabeling, that applies to samples before sending them to the remote endpoint. This could be used to limit which samples are sent.

    ```yaml
    remote_write:
      - url: "http://localhost:9201/write"
        write_relabel_configs:
          - source_labels: [job]
            regex: 'prometheus'
            action: keep
    ```


Metrics sent to the http endpoint will be put by default under the `prometheus.metrics` prefix with their labels under `prometheus.labels`. A basic configuration would look like:

```yaml
- module: prometheus
  metricsets: ["remote_write"]
  host: "localhost"
  port: "9201"
```

Also consider using secure settings for the server, configuring the module with TLS/SSL as shown:

```yaml
- module: prometheus
  metricsets: ["remote_write"]
  host: "localhost"
  ssl.certificate: "/etc/pki/server/cert.pem"
  ssl.key: "/etc/pki/server/cert.key"
  port: "9201"
```

and on Prometheus side:

```yaml
remote_write:
  - url: "https://localhost:9201/write"
    tls_config:
        cert_file: "/etc/prometheus/my_key.pem"
        key_file: "/etc/prometheus/my_key.key"
        # Disable validation of the server certificate.
        #insecure_skip_verify: true
```


## Histograms and types [_histograms_and_types_2]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


```yaml
metricbeat.modules:
- module: prometheus
  metricsets: ["remote_write"]
  host: "localhost"
  port: "9201"
  use_types: true
  rate_counters: true
  period: 60s
```

`use_types` parameter (default: false) enables a different layout for metrics storage, leveraging Elasticsearch types, including [histograms](elasticsearch://reference/elasticsearch/mapping-reference/histogram.md).

`rate_counters` parameter (default: false) enables calculating a rate out of Prometheus counters. When enabled, Metricbeat stores the counter increment since the last collection. This metric should make some aggregations easier and with better performance. This parameter can only be enabled in combination with `use_types`.

`period` parameter (default: 60s) configures the timeout of internal cache, which stores counter values in order to calculate rates between consecutive fetches. The parameter will be validated and all values lower than 60sec will be reset to the default value.

Note that by default prometheus pushes data with the interval of 60s (in remote write). In case that prometheus push rate is changed, the `period` parameter needs to be configured accordingly.

When `use_types` and `rate_counters` are enabled, metrics are stored like this:

```json
{
    "prometheus": {
        "labels": {
            "instance": "172.27.0.2:9090",
            "job": "prometheus"
        },
        "prometheus_target_interval_length_seconds_count": {
            "counter": 1,
            "rate": 0
        },
        "prometheus_target_interval_length_seconds_sum": {
            "counter": 15.000401344,
            "rate": 0
        }
        "prometheus_tsdb_compaction_chunk_range_seconds_bucket": {
            "histogram": {
                "values": [50, 300, 1000, 4000, 16000],
                "counts": [10, 2, 34, 7]
            }
        }
    },
}
```


### Types' patterns [_types_patterns]

Unlike `collector` metricset, `remote_write` receives metrics in raw format from the prometheus server. In this, the module has to internally use a heuristic in order to identify efficiently the type of each raw metric. For these purpose some name patterns are used in order to identify the type of each metric. The default patterns are the following:

1. `_total` suffix: the metric is of Counter type
2. `_sum` suffix: the metric is of Counter type
3. `_count` suffix: the metric is of Counter type
4. `_bucket` suffix and `le` in labels: the metric is of Histogram type

Everything else is handled as a Gauge. In addition there is no special handling for Summaries so it is expected that Summary’s quantiles are handled as Gauges and Summary’s sum and count as Counters.

Users have the flexibility to add their own patterns using the following configuration:

```yaml
metricbeat.modules:
- module: prometheus
  metricsets: ["remote_write"]
  host: "localhost"
  port: "9201"
  types_patterns:
    counter_patterns: ["_my_counter_suffix"]
    histogram_patterns: ["_my_histogram_suffix"]
```

The configuration above will consider metrics with names that match `_my_counter_suffix` as Counters and those that match `_my_histogram_suffix` (and have `le` in their labels) as Histograms.

To match only specific metrics, anchor the start and the end of the regexp of each metric:

* the caret `^` matches the beginning of a text or line,
* the dollar sign `$` matches the end of a text.

```yaml
metricbeat.modules:
- module: prometheus
  metricsets: ["remote_write"]
  host: "localhost"
  port: "9201"
  types_patterns:
    histogram_patterns: ["^my_histogram_metric$"]
```

Note that when using `types_patterns`, the provided patterns have higher priority than the default patterns. For instance if `_histogram_total` is a defined histogram pattern, then a metric like `network_bytes_histogram_total` will be handled as a histogram, even if it has the suffix `_total` which is a default pattern for counters.
