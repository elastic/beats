The PLC4X module reads tags from programmable logic controllers (PLCs) using [Apache PLC4X](https://plc4x.apache.org/).

The default metricset is `tag`.


## Compatibility [_compatibility_plc4x]

The module uses PLC4X 0.13.1 and supports the Modbus TCP driver (`modbus-tcp`). It is tested against a Modbus TCP server built with pymodbus 3.16.1.

Modbus holding registers and input registers are supported with all numeric and string data types. Modbus coils and discrete inputs are not supported yet, because PLC4X 0.13 decodes them incorrectly; the module rejects such tags.


## Configuration [_configuration_plc4x]

Each entry in `hosts` is a PLC4X connection string, for example `modbus-tcp://plc.example.com:502?unit-identifier=1`. Tag addresses use the PLC4X Modbus syntax `<area>:<address>:<data type>`, for example `holding-register:1:REAL`. Addresses are 1-based, so `holding-register:1` is the first holding register.
