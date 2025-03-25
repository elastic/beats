---
navigation_title: "PgSQL"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-pgsql-options.html
---

# Capture PgSQL traffic [packetbeat-pgsql-options]


The `pgsql` sections of the `packetbeat.yml` config file specifies configuration options for the PgSQL protocols.

```yaml
packetbeat.protocols:

- type: pgsql
  ports: [5432]
```

## Configuration options [_configuration_options_9]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `max_rows` [_max_rows_2]

The maximum number of rows from the SQL message to publish to Elasticsearch. The default is 10 rows.


### `max_row_length` [_max_row_length_2]

The maximum length in bytes of a row from the SQL message to publish to Elasticsearch. The default is 1024 bytes.



