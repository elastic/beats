---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-statsd.html
---

# Statsd module [metricbeat-module-statsd]

The `statsd` module is a Metricbeat module which spawns a UDP server and listens for metrics in StatsD compatible format.


## Metric types [_metric_types]

The module supports the following types of metrics:

**Counter (c)**
:   Measurement which accumulates over period of time until flushed (value set to 0).

**Gauge (g)**
:   Measurement which can increase, decrease or be set to a value.

**Timer (ms)**
:   Time measurement (in milliseconds) of an event.

**Histogram (h)**
:   Time measurement, alias for timer.

**Set (s)**
:   Measurement which counts unique occurrences until flushed (value set to 0).


## Supported tag extensions [_supported_tag_extensions]

Example of tag styles supported by the `statsd` module:

[DogStatsD](https://docs.datadoghq.com/developers/dogstatsd/datagram_shell/?tab=metrics#the-dogstatsd-protocol)

`<metric name>:<value>|<type>|@samplerate|#<k>:<v>,<k>:<v>`

[InfluxDB](https://github.com/influxdata/telegraf/blob/master/plugins/inputs/statsd/README.md#influx-statsd)

`<metric name>,<k>=<v>,<k>=<v>:<value>|<type>|@samplerate`

[Graphite_1.1.x](https://graphite.readthedocs.io/en/latest/tags.html#graphite-tag-support)

`<metric name>;<k>=<v>;<k>=<v>:<value>|<type>|@samplerate`


## Module-specific configuration notes [_module_specific_configuration_notes_20]

The `statsd` module has these additional config options:

**`ttl`**
:   It defines how long a metric will be reported after it was last recorded. Irrespective of the given ttl, metrics will be reported at least once. A ttl of zero means metrics will never expire.

**`statsd.mapping`**
:   It defines how metrics will mapped from the original metric label to the event json. Here’s an example configuration:

```yaml
statsd.mappings:
  - metric: 'ti_failures' <1>
    value:
      field: task_failures <2>
  - metric: '<job_name>_start' <1>
    labels:
      - attr: job_name <3>
        field: job_name <4>
    value:
      field: started <2>
```

1. `metric`, required: the label key of the metric in statsd, either as a exact match string or as a template with named label placeholder in the format `<label_placeholder>`
2. `value.field`, required: field name where to save the metric value in the event json
3. `label[].attr`, required when using named label placeholder: reference to the named label placeholder defined in `metric`
4. `label[].field`, required when using named label placeholder field name where to save the named label placeholder value from the template in the event json




