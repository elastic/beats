---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-gcp-metrics.html
---

# Google Cloud Platform metrics metricset [metricbeat-metricset-gcp-metrics]

Operations monitoring provides visibility into the performance, uptime, and overall health of cloud-powered applications. It collects metrics, events, and metadata from different services from Google Cloud. This metricset is to collect monitoring metrics from Google Cloud using `ListTimeSeries` API. The full list of metric types that Google Cloud monitoring supports can be found in [Google Cloud Metrics](https://cloud.google.com/monitoring/api/metrics_gcp#gcp).

Each monitoring metric from Google Cloud has a sample period and/or ingest delay. Sample period is the time interval between consecutive data points for metrics that are written periodically. Ingest delay represents the time for data points older than this value are guaranteed to be available to read. Sample period and ingest delay are obtained from making [ListMetricDescriptors API](https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors/list) call.


## Metricset config and parameters [_metricset_config_and_parameters]

* **aligner**: A single string with which aggregation operation need to be applied onto time series data for ListTimeSeries API. If it’s not given, default aligner is `ALIGN_NONE`. Google Cloud also supports `ALIGN_DELTA`, `ALIGN_RATE`, `ALIGN_MIN`, `ALIGN_MAX`, `ALIGN_MEAN`, `ALIGN_COUNT`, `ALIGN_SUM` etc. Please see [Aggregation Aligner](https://cloud.google.com/monitoring/api/ref_v3/rpc/google.monitoring.v3#aligner) for the full list of aligners.
* **metric_types**: Required, a list of metric type strings, or a list of metric type prefixes. For example, `instance/cpu` is the prefix for metric type `instance/cpu/usage_time`, `instance/cpu/utilization` etc Each call of the `ListTimeSeries` API can return any number of time series from a single metric type. Metric type is to used for identifying a specific time series.
* **service**: Required, the name of the service for related metrics. This should be a valid Google Cloud service name. Service names may be viewed from the corresponding page from [GCP Metrics list documentation](https://cloud.google.com/monitoring/api/metrics). The `service` field is used to compute the GCP Metric prefix, unless `service_metric_prefix` is set.
* **service_metric_prefix**: A string containing the full Metric prefix as specified in the GCP documentation. All metrics from GCP Monitoring API require a prefix. When `service_metric_prefix` is empty, the prefix default to a value computed using the `service` value: `<service>.googleapis.com/`. This default works for any services under "Google Cloud metrics", but does not work for other services (`kubernetes` aka GKE for example). This option allow to override the default and specify an arbitrary metric prefix.
* **location_label**: Use this option to specify the resource label that identifies the location (such as zone or region) for a Google Cloud service when filtering metrics. For example, labels like `resource.label.location` or `resource.label.zone` are used by Google Cloud to represent the region or zone of a resource. This is an optional configuration for the user.


## Example Configuration [_example_configuration_26]

* `metrics` metricset is enabled to collect metrics from all zones under `europe-west1-c` region in `elastic-observability` project. Two sets of metrics are specified: first one is to collect CPU usage time and utilization with aggregation aligner ALIGN_MEAN; second one is to collect uptime with aggregation aligner ALIGN_SUM. These metric types all have 240 seconds ingest delay time and 60 seconds sample period. With `period` specified as `300s` in the config below, Metricbeat will collect compute metrics from Google Cloud every 5-minute with given aggregation aligner applied for each metric type.

    ```yaml
    - module: gcp
      metricsets:
        - metrics
      zone: "europe-west1-c"
      project_id: elastic-observability
      credentials_file_path: "your JSON credentials file path"
      exclude_labels: false
      period: 300s
      metrics:
        - aligner: ALIGN_MEAN
          service: compute
          metric_types:
            - "instance/cpu/usage_time"
            - "instance/cpu/utilization"
        - aligner: ALIGN_SUM
          service: compute
          metric_types:
            - "instance/uptime"
    ```

* `metrics` metricset is enabled to collect metrics from all zones under `europe-west1-c` region in `elastic-observability` project. Two sets of metrics are specified: first one is to collect CPU usage time and utilization with aggregation aligner ALIGN_MEAN; second one is to collect uptime with aggregation aligner ALIGN_SUM. These metric types all have 240 seconds ingest delay time and 60 seconds sample period. With `period` specified as `60s` in the config below, Metricbeat will collect compute metrics from Google Cloud every minute with no aggregation. This case, the aligners specified in the configuration will be ignored.

    ```yaml
    - module: gcp
      metricsets:
        - metrics
      zone: "europe-west1-c"
      project_id: elastic-observability
      credentials_file_path: "your JSON credentials file path"
      exclude_labels: false
      period: 60s
      metrics:
        - aligner: ALIGN_MEAN
          service: compute
          metric_types:
            - "instance/cpu/usage_time"
            - "instance/cpu/utilization"
        - aligner: ALIGN_SUM
          service: compute
          metric_types:
            - "instance/uptime"
    ```

* `metrics` metricset is enabled to collect metrics from all zones under `europe-west1-c` region in `elastic-observability` project. One set of metrics will be collected: core usage time for containers in GCP GKE. Note that the is required to use `service_metric_prefix` to override the default metric prefix, as for GKE metrics the required prefix is `kubernetes.io/`

    ```yaml
    - module: gcp
      metricsets:
        - metrics
      zone: "europe-west1-c"
      project_id: elastic-observability
      credentials_file_path: "your JSON credentials file path"
      exclude_labels: false
      period: 1m
      metrics:
        - service: gke
          service_metric_prefix: kubernetes.io/
          metric_types:
            - "container/cpu/core_usage_time"
    ```

* `metrics` metricset is enabled to collect metrics from region `us-east4` in `elastic-observability` project. The metric, number of replicas of the prediction model is collected from a new GCP service `aiplatform`. Since its a new service which is not supported by default in this metricset, the user provides the servicelabel (resource.label.location), for which user wants to filter the incoming data

    ```yaml
    - module: gcp
      metricsets:
        - metrics
      project_id: "elastic-observability"
      credentials_json: "your JSON credentials"
      exclude_labels: false
      period: 1m
      location_label: "resource.label.location" # This is an optional configuration
      regions:
      - us-east4
      metrics:
        - service: aiplatform
          metric_types:
              - "prediction/online/replicas"
    ```


## Fields [_fields_106]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-gcp.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "cloud": {
        "account": {
            "id": "elastic-observability",
            "name": "elastic-observability"
        },
        "instance": {
            "id": "4049989596327614796",
            "name": "nchaulet-loadtest-horde-master"
        },
        "machine": {
            "type": "n1-standard-8"
        },
        "provider": "gcp"
    },
    "cloud.availability_zone": "us-central1-a",
    "cloud.region": "us-central1",
    "event": {
        "dataset": "gcp.metrics",
        "duration": 115000,
        "module": "gcp"
    },
    "gcp": {
        "labels": {},
        "metrics": {
            "instance": {
                "uptime_total": {
                    "value": 791820
                }
            }
        }
    },
    "host": {
        "id": "4049989596327614796",
        "name": "nchaulet-loadtest-horde-master"
    },
    "metricset": {
        "name": "metrics",
        "period": 10000
    },
    "service": {
        "type": "gcp"
    }
}
```


