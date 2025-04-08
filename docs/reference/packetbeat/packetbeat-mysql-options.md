---
navigation_title: "MySQL"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-mysql-options.html
---

# Capture MySQL traffic [packetbeat-mysql-options]


The `mysql` section of the `packetbeat.yml` config file specifies configuration options for the MySQL protocols.

```yaml
packetbeat.protocols:

- type: mysql
  ports: [3306,3307]
```

## Configuration options [_configuration_options_8]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `max_rows` [_max_rows]

The maximum number of rows from the SQL message to publish to Elasticsearch. The default is 10 rows.


### `max_row_length` [_max_row_length]

The maximum length in bytes of a row from the SQL message to publish to Elasticsearch. The default is 1024 bytes.



## `statement_timeout` [_statement_timeout]

The duration for which prepared statements are cached after their last use. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". The default is `1h`.


