---
navigation_title: "Cassandra"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-cassandra.html
---

# Capture Cassandra traffic [configuration-cassandra]


The following settings are specific to the Cassandra protocol. Here is a sample configuration for the `cassandra` section of the `packetbeat.yml` config file:

```yaml
packetbeat.protocols:
- type: cassandra
  send_request_header: true
  send_response_header: true
  compressor: "snappy"
  ignored_ops: ["SUPPORTED","OPTIONS"]
```

## Configuration options [_configuration_options_6]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `send_request_header` [_send_request_header]

If this option is enabled, the raw message of the response (`cassandra_request.request_headers` field) is sent to Elasticsearch. The default is true. enable `send_request` first before enable this option.


### `send_response_header` [_send_response_header]

If this option is enabled, the raw message of the response (`cassandra_response.response_headers` field) is included in published events. The default is true. enable `send_response` first before enable this option.


### `ignored_ops` [_ignored_ops]

This option indicates which Operator/Operators captured will be ignored. currently support: `ERROR` ,`STARTUP` ,`READY` ,`AUTHENTICATE` ,`OPTIONS` ,`SUPPORTED` , `QUERY` ,`RESULT` ,`PREPARE` ,`EXECUTE` ,`REGISTER`  ,`EVENT` , `BATCH` ,`AUTH_CHALLENGE`,`AUTH_RESPONSE` ,`AUTH_SUCCESS` .


### `compressor` [_compressor]

Configures the default compression algorithm being used to uncompress compressed frames by name. Currently only `snappy` is can be configured. By default no compressor is configured.



