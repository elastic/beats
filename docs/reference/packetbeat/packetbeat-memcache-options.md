---
navigation_title: "Memcache"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-memcache-options.html
---

# Capture Memcache traffic [packetbeat-memcache-options]


The `memcache` section of the `packetbeat.yml` config file specifies configuration options for the memcache protocol. Here is a sample configuration section for memcache:

```yaml
packetbeat.protocols:
- type: memcache
  ports: [11211]
  parseunknown: false
  maxvalues: 0
  maxbytespervalue: 100
  transaction_timeout: 200
  udptransactiontimeout: 200
```

## Configuration options [_configuration_options_7]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `parseunknown` [_parseunknown]

When this option is enabled, it forces the memcache text protocol parser to accept unknown commands.

::::{note}
The unknown commands MUST NOT contain a data part.
::::



### `maxvalues` [_maxvalues]

The maximum number of values to store in the message (multi-get). All values will be base64 encoded.

The possible settings for this option are:

* `maxvalue: -1`, which stores all values (text based protocol multi-get)
* `maxvalue: 0`, which stores no values (default)
* `maxvalue: N`, which stores up to N values


### `maxbytespervalue` [_maxbytespervalue]

The maximum number of bytes to be copied for each value element.

::::{note}
Values will be base64 encoded, so the actual size in the JSON document will be 4 times the value that you specify for `maxbytespervalue`.
::::



### `udptransactiontimeout` [_udptransactiontimeout]

The transaction timeout in milliseconds. The defaults is 10000 milliseconds.

::::{note}
Quiet messages in UDP binary protocol get responses only if there is an error. The memcache protocol analyzer will wait for the number of milliseconds specified by `udptransactiontimeout` before publishing quiet messages. Non-quiet messages or quiet requests with an error response are published immediately.
::::




