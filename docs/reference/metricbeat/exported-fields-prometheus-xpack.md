---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-prometheus-xpack.html
---

# Prometheus typed metrics fields [exported-fields-prometheus-xpack]

Stats scraped from a Prometheus endpoint.

**`prometheus.*.value`**
:   Prometheus gauge metric

type: object


**`prometheus.*.counter`**
:   Prometheus counter metric

type: object


**`prometheus.*.rate`**
:   Prometheus rated counter metric

type: object


**`prometheus.*.histogram`**
:   Prometheus histogram metric - release: ga

type: object


