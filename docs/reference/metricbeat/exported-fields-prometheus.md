---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-prometheus.html
---

# Prometheus fields [exported-fields-prometheus]

Stats scraped from a Prometheus endpoint.

**`metrics_count`**
:   Number of metrics per Elasticsearch document.

type: long


**`prometheus.labels.*`**
:   Prometheus metric labels

type: object


**`prometheus.metrics.*`**
:   Prometheus metric

type: object


**`prometheus.query.*`**
:   Prometheus value resulted from PromQL

type: object



## query [_query_3]

query metricset


## remote_write [_remote_write]

remote write metrics from Prometheus server

