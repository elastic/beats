---
navigation_title: "AMQP"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-amqp-options.html
---

# Capture AMQP traffic [packetbeat-amqp-options]


The `amqp` section of the `packetbeat.yml` config file specifies configuration options for the AMQP 0.9.1 protocol. Here is a sample configuration:

```yaml
packetbeat.protocols:
- type: amqp
  ports: [5672]
  max_body_length: 1000
  parse_headers: true
  parse_arguments: false
  hide_connection_information: true
```

## Configuration options [_configuration_options_5]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `max_body_length` [_max_body_length]

The maximum size in bytes of the message displayed in the request or response fields. Messages that are bigger than the specified size are truncated. Use this option to avoid publishing huge messages when [`send_request`](/reference/packetbeat/common-protocol-options.md#send-request-option) or [`send_request`](/reference/packetbeat/common-protocol-options.md#send-request-option) is enabled. The default is 1000 bytes.


### `parse_headers` [_parse_headers]

If set to true, Packetbeat parses the additional arguments specified in the headers field of a message. Those arguments are key-value pairs that specify information such as the content type of the message or the message priority. The default is true.


### `parse_arguments` [_parse_arguments]

If set to true, Packetbeat parses the additional arguments specified in AMQP methods. Those arguments are key-value pairs specified by the user and can be of any length. The default is true.


### `hide_connection_information` [_hide_connection_information]

If set to false, the connection layer methods of the protocol are also displayed, such as the opening and closing of connections and channels by clients, or the quality of service negotiation. The default is true.



