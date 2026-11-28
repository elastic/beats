---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-plc4x.html
applies_to:
  stack: beta
  serverless: beta
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# PLC4X fields [exported-fields-plc4x]

PLC4X module

## plc4x [_plc4x]

Values read from PLCs with Apache PLC4X.

## tag [_tag]

```{applies_to}
stack: beta
```

The value of one PLC tag.

**`plc4x.tag.address`**
:   The tag address, as configured, for example holding-register:1:REAL.

    type: keyword


**`plc4x.tag.name`**
:   The name configured for the tag.

    type: keyword


**`plc4x.tag.data_type`**
:   The IEC 61131-3 data type of the value, for example REAL or INT. Lists and structures are reported as List and Struct and are not indexed.

    type: keyword


**`plc4x.tag.response_code`**
:   The PLC4X response code of the read, for example OK or INVALID_ADDRESS.

    type: keyword


## value [_value]

The value of the tag. Exactly one of the typed fields is set, depending on the data type.

**`plc4x.tag.value.double`**
:   REAL and LREAL values, and unsigned 64-bit values above the range of a long.

    type: double


**`plc4x.tag.value.long`**
:   Integer and bit string values.

    type: long


**`plc4x.tag.value.boolean`**
:   BOOL values.

    type: boolean


**`plc4x.tag.value.keyword`**
:   CHAR, WCHAR, STRING and WSTRING values.

    type: keyword


**`plc4x.tag.value.date`**
:   DATE and DATE_AND_TIME values.

    type: date


