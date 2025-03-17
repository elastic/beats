---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-gcp-gke.html
---

# Google Cloud Platform gke metricset [metricbeat-metricset-gcp-gke]

`gke` metricset is designed for collecting metrics from [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine). Google Cloud Monitoring supports Google Kubernetes Engine metrics, as listed in [Google Cloud Monitoring Kubernetes metrics](https://cloud.google.com/monitoring/api/metrics_kubernetes).

This metricset collects all GA Kubernetes metrics from Google Cloud Monitoring APIs. It leverages under the hood the `metrics` metricset. The field names are aligned to [Beats naming conventions](/extend/event-conventions.md) with minor modifications to their GCP metrics name counterpart.

We recommend users to define `period: 1m` for this metricset because in Google Cloud, GKE monitoring metrics are sampled every 60 seconds. Some of the metrics have an ingest delay up to 240 seconds.


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_14]

None


## Configuration example [_configuration_example_22]

```yaml
- module: gcp
  metricsets:
    - gke
  project_id: "your project id"
  credentials_file_path: "your JSON credentials file path"
  exclude_labels: false
  period: 1m
```

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields_104]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-gcp.md) section.


