---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-trans_measurements.html
---

# Measurements (Transactions) fields [exported-fields-trans_measurements]

These fields contain measurements related to the transaction.

**`bytes_in`**
:   The number of bytes of the request. Note that this size is the application layer message length, without the length of the IP or TCP headers.

type: alias

alias to: source.bytes


**`bytes_out`**
:   The number of bytes of the response. Note that this size is the application layer message length, without the length of the IP or TCP headers.

type: alias

alias to: destination.bytes


