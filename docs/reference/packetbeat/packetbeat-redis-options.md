---
navigation_title: "Redis"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-redis-options.html
---

# Capture Redis traffic [packetbeat-redis-options]


The Redis protocol has several specific configuration options. Here is a sample configuration for the `redis` section of the `packetbeat.yml` config file:

```yaml
packetbeat.protocols:
- type: redis
  ports: [6379]
  queue_max_bytes: 1048576
  queue_max_messages: 20000
```

## Configuration options [_configuration_options_13]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `queue_max_bytes` and `queue_max_messages` [_queue_max_bytes_and_queue_max_messages]

In order for request/response correlation to work, Packetbeat needs to store requests in memory until a response is received. These settings impose a limit on the number of bytes (`queue_max_bytes`) and number of requests (`queue_max_messages`) that can be stored. These limits are per-connection. The default is to queue up to 1MB or 20.000 requests per connection, which allows to use request pipelining while at the same time limiting the amount of memory consumed by replication sessions.



