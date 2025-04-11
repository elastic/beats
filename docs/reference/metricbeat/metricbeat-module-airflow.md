---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-airflow.html
---

# Airflow module [metricbeat-module-airflow]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This module collects metrics from [Airflow metrics](https://airflow.apache.org/docs/apache-airflow/stable/logging-monitoring/metrics.html). It runs a statsd server where airflow will send metrics to. The default metricset is `statsd`.


## Compatibility [_compatibility_6]

The Airflow module is tested with Airflow 2.1.0. It should work with version 2.0.0 and later.


## Usage [_usage_2]

The Airflow module requires [Statsd](/reference/metricbeat/metricbeat-module-statsd.md) to receive statsd metrics. Refer to the link for instructions about how to use statsd.

Add the following lines to your Airflow configuration file e.g. `airflow.cfg` ensuring `statsd_prefix` is left empty and replace `%METRICBEAT_HOST%` with the address where metricbeat is running:

```
[metrics]
statsd_on = True
statsd_host = %METRICBEAT_HOST%
statsd_port = 8126
statsd_prefix =
```


## Example configuration [_example_configuration_3]

The Airflow module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: airflow
  host: "localhost"
  port: "8126"
  #ttl: "30s"
  metricsets: [ 'statsd' ]
```


## Metricsets [_metricsets_4]

The following metricsets are available:

* [statsd](/reference/metricbeat/metricbeat-metricset-airflow-statsd.md)


