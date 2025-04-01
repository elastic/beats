---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-thrift.html
---

# Thrift-RPC fields [exported-fields-thrift]

Thrift-RPC specific event fields.

**`thrift.params`**
:   The RPC method call parameters in a human readable format. If the IDL files are available, the parameters use names whenever possible. Otherwise, the IDs from the message are used.


**`thrift.service`**
:   The name of the Thrift-RPC service as defined in the IDL files.


**`thrift.return_value`**
:   The value returned by the Thrift-RPC call. This is encoded in a human readable format.


**`thrift.exceptions`**
:   If the call resulted in exceptions, this field contains the exceptions in a human readable format.


