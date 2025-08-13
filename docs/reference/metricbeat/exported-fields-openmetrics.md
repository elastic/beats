---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-openmetrics.html
applies_to:
  stack: beta
---

% This file is generated! See scripts/generate_fields_docs.py

# Openmetrics fields [exported-fields-openmetrics]

Openmetrics module

## openmetrics [_openmetrics]

```{applies_to}
stack: beta
```

`openmetrics` contains metrics from endpoints that are following Openmetrics format.

**`openmetrics.help`**
:   Brief description of the MetricFamily

    type: keyword


**`openmetrics.type`**
:   Metric type

    type: keyword


**`openmetrics.unit`**
:   Metric unit

    type: keyword


**`openmetrics.labels.*`**
:   Openmetrics metric labels

    type: object


**`openmetrics.metrics.*`**
:   Openmetrics metric

    type: object


**`openmetrics.exemplar.*`**
:   Openmetrics exemplars

    type: object


**`openmetrics.exemplar.labels.*`**
:   Openmetrics metric exemplar labels

    type: object


