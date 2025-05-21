::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the ingest_pipeline metricset of the module elasticsearch.

Collects metrics on ingest pipeline executions, with processor-level granularity.


## Processor-level metrics sampling [_processor_level_metrics_sampling]

Processor-level metrics can produce a high volume of data, so the default behavior is to collect those metrics less frequently than the `period` for pipeline-level metrics, by applying a sampling strategy. By default, the processor-level metrics will be collected during 25% of the time. This can be configured with the `ingest.processor_sample_rate` setting:


## Configuration example [_configuration_example_19]

```yaml
- module: elasticsearch
  period: 10s
  metricsets:
    - ingest_pipeline
  ingest.processor_sample_rate: 0.1 # decrease to 10% of fetches
```

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
