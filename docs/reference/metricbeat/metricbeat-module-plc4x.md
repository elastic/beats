---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-plc4x.html
applies_to:
  stack: beta
  serverless: beta
---

% This file is generated! See metricbeat/scripts/mage/docs_collector.go

# PLC4X module [metricbeat-module-plc4x]

The PLC4X module reads tags from programmable logic controllers (PLCs) using [Apache PLC4X](https://plc4x.apache.org/).

The default metricset is `tag`.


## Compatibility [_compatibility_plc4x]

The module uses PLC4X 0.13.1 and supports the Modbus TCP driver (`modbus-tcp`). It is tested against a Modbus TCP server built with pymodbus 3.16.1.

Modbus holding registers and input registers are supported with all numeric and string data types. Modbus coils and discrete inputs are not supported yet, because PLC4X 0.13 decodes them incorrectly; the module rejects such tags.


## Configuration [_configuration_plc4x]

Each entry in `hosts` is a PLC4X connection string, for example `modbus-tcp://plc.example.com:502?unit-identifier=1`. Tag addresses use the PLC4X Modbus syntax `<area>:<address>:<data type>`, for example `holding-register:1:REAL`. Addresses are 1-based, so `holding-register:1` is the first holding register.


## Example configuration [_example_configuration]

The PLC4X module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: plc4x
  metricsets: ["tag"]
  period: 10s
  # PLC4X connection strings. Only the modbus-tcp driver is supported, for
  # example modbus-tcp://plc.example.com:502?unit-identifier=1.
  hosts: ["modbus-tcp://localhost:502"]
  # Timeout for connecting and for each read request.
  #timeout: 10s

  # PLC tags to read, in PLC4X address syntax <area>:<address>:<data type>, for
  # example holding-register:1:REAL or input-register:5:UINT. Addresses are
  # 1-based. Modbus coils and discrete inputs are not supported yet. The
  # optional name is added to the event.
  plc_tags:
    - address: "holding-register:1:REAL"
      name: "temperature"
```


## Metricsets [_metricsets]

The following metricsets are available:

* [tag](/reference/metricbeat/metricbeat-metricset-plc4x-tag.md)  {applies_to}`stack: beta`
