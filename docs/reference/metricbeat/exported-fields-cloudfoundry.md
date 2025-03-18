---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-cloudfoundry.html
---

# Cloudfoundry fields [exported-fields-cloudfoundry]

Cloud Foundry module


## cloudfoundry [_cloudfoundry]

**`cloudfoundry.type`**
:   The type of event from Cloud Foundry. Possible values include *container*, *counter* and *value*.

type: keyword



## app [_app]

The application the metric is associated with.

**`cloudfoundry.app.id`**
:   The ID of the application.

type: keyword



## container [_container_2]

`container` contains container metrics from Cloud Foundry.

**`cloudfoundry.container.instance_index`**
:   Index of the instance the metric belongs to.

type: long


**`cloudfoundry.container.cpu.pct`**
:   CPU usage percentage.

type: scaled_float


**`cloudfoundry.container.memory.bytes`**
:   Bytes of used memory.

type: long


**`cloudfoundry.container.memory.quota.bytes`**
:   Bytes of available memory.

type: long


**`cloudfoundry.container.disk.bytes`**
:   Bytes of used storage.

type: long


**`cloudfoundry.container.disk.quota.bytes`**
:   Bytes of available storage.

type: long



## counter [_counter_2]

`counter` contains counter metrics from Cloud Foundry.

**`cloudfoundry.counter.name`**
:   The name of the counter.

type: keyword


**`cloudfoundry.counter.delta`**
:   The difference between the last time the counter event occurred.

type: long


**`cloudfoundry.counter.total`**
:   The total value for the counter.

type: long



## value [_value_2]

`value` contains counter metrics from Cloud Foundry.

**`cloudfoundry.value.name`**
:   The name of the value.

type: keyword


**`cloudfoundry.value.unit`**
:   The unit of the value.

type: keyword


**`cloudfoundry.value.value`**
:   The value of the value.

type: float


