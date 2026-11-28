---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-opcua.html
applies_to:
  stack: beta
  serverless: beta
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# OPC UA fields [exported-fields-opcua]

OPC UA module

## opcua [_opcua]

Values read from OPC UA servers.

## node [_node]

```{applies_to}
stack: beta
```

The value of one OPC UA node, read in one request together with the other configured nodes.

**`opcua.node.id`**
:   The node id, as configured, for example ns=3;s=StepUp.

    type: keyword


**`opcua.node.label`**
:   The label configured for the node.

    type: keyword


**`opcua.node.name`**
:   The display name of the node, read from the server.

    type: keyword


**`opcua.node.data_type`**
:   The OPC UA built-in data type of the value, for example Double or Int32. Arrays end in [] and are not indexed.

    type: keyword


## value [_value]

The value of the node. Exactly one of the typed fields is set, depending on the data type.

**`opcua.node.value.double`**
:   Float and Double values, and UInt64 values above the range of a long.

    type: double


**`opcua.node.value.long`**
:   Integer values and status codes.

    type: long


**`opcua.node.value.boolean`**
:   Boolean values.

    type: boolean


**`opcua.node.value.keyword`**
:   String, LocalizedText, QualifiedName, NodeId and Guid values.

    type: keyword


**`opcua.node.value.date`**
:   DateTime values.

    type: date


## status [_status]

The status code of the value.

**`opcua.node.status.code`**
:   The numeric status code.

    type: long


**`opcua.node.status.name`**
:   The name of the status code, for example Good or StatusBadNodeIDUnknown.

    type: keyword


**`opcua.node.status.severity`**
:   The severity of the status code: good, uncertain or bad.

    type: keyword


**`opcua.node.source_timestamp`**
:   The time the value was produced by its source.

    type: date


**`opcua.node.server_timestamp`**
:   The time the server received or updated the value.

    type: date


