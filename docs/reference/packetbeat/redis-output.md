---
navigation_title: "Redis"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/redis-output.html
---

# Configure the Redis output [redis-output]


The Redis output inserts the events into a Redis list or a Redis channel. This output plugin is compatible with the [Redis input plugin](logstash-docs-md://lsr/plugins-inputs-redis.md) for Logstash.

To use this output, edit the Packetbeat configuration file to disable the {{es}} output by commenting it out, and enable the Redis output by adding `output.redis`.

Example configuration:

```yaml
output.redis:
  hosts: ["localhost"]
  password: "my_password"
  key: "packetbeat"
  db: 0
  timeout: 5
```

## Compatibility [_compatibility_3]

This output is expected to work with all Redis versions between 3.2.4 and 5.0.8. Other versions might work as well, but are not supported.


## Configuration options [_configuration_options_19]

You can specify the following `output.redis` options in the `packetbeat.yml` config file:

### `enabled` [_enabled_7]

The enabled config is a boolean setting to enable or disable the output. If set to false, the output is disabled.

The default value is `true`.


### `hosts` [_hosts_2]

The list of Redis servers to connect to. If load balancing is enabled, the events are distributed to the servers in the list. If one server becomes unreachable, the events are distributed to the reachable servers only. You can define each Redis server by specifying `HOST` or `HOST:PORT`. For example: `"192.15.3.2"` or `"test.redis.io:12345"`. If you don’t specify a port number, the value configured by `port` is used. Configure each Redis server with an `IP:PORT` pair or with a `URL`. For example: `redis://localhost:6379` or `rediss://localhost:6379`. URLs can include a server-specific password. For example: `redis://:password@localhost:6379`. The `redis` scheme will disable the `ssl` settings for the host, while `rediss` will enforce TLS.  If `rediss` is specified and no `ssl` settings are configured, the output uses the system certificate store.


### `index` [_index_2]

The index name added to the events metadata for use by Logstash. The default is "packetbeat".


### `key` [key-option-redis]

The name of the Redis list or channel the events are published to. If not configured, the value of the `index` setting is used.

You can set the key dynamically by using a format string to access any event field. For example, this configuration uses a custom field, `fields.list`, to set the Redis list key. If `fields.list` is missing, `fallback` is used:

```yaml
output.redis:
  hosts: ["localhost"]
  key: "%{[fields.list]:fallback}"
```

::::{tip}
To learn how to add custom fields to events, see the [`fields`](/reference/packetbeat/configuration-general-options.md#libbeat-configuration-fields) option.
::::


See the [`keys`](#keys-option-redis) setting for other ways to set the key dynamically.


### `keys` [keys-option-redis]

An array of key selector rules. Each rule specifies the `key` to use for events that match the rule. During publishing, Packetbeat uses the first matching rule in the array. Rules can contain conditionals, format string-based fields, and name mappings. If the `keys` setting is missing or no rule matches, the [`key`](#key-option-redis) setting is used.

Rule settings:

**`index`**
:   The key format string to use. If this string contains field references, such as `%{[fields.name]}`, the fields must exist, or the rule fails.

**`mappings`**
:   A dictionary that takes the value returned by `key` and maps it to a new name.

**`default`**
:   The default string value to use if `mappings` does not find a match.

**`when`**
:   A condition that must succeed in order to execute the current rule. All the [conditions](/reference/packetbeat/defining-processors.md#conditions) supported by processors are also supported here.

Example `keys` settings:

```yaml
output.redis:
  hosts: ["localhost"]
  key: "default_list"
  keys:
    - key: "info_list"   # send to info_list if `message` field contains INFO
      when.contains:
        message: "INFO"
    - key: "debug_list"  # send to debug_list if `message` field contains DEBUG
      when.contains:
        message: "DEBUG"
    - key: "%{[fields.list]}"
      mappings:
        http: "frontend_list"
        nginx: "frontend_list"
        mysql: "backend_list"
```


### `password` [_password_3]

The password to authenticate with. The default is no authentication.


### `db` [_db]

The Redis database number where the events are published. The default is 0.


### `datatype` [_datatype]

The Redis data type to use for publishing events.If the data type is `list`, the Redis RPUSH command is used and all events are added to the list with the key defined under `key`. If the data type `channel` is used, the Redis `PUBLISH` command is used and means that all events are pushed to the pub/sub mechanism of Redis. The name of the channel is the one defined under `key`. The default value is `list`.


### `codec` [_codec_2]

Output codec configuration. If the `codec` section is missing, events will be json encoded.

See [Change the output codec](/reference/packetbeat/configuration-output-codec.md) for more information.


### `worker` or `workers` [_worker_or_workers_2]

The number of workers to use for each host configured to publish events to Redis. Use this setting along with the `loadbalance` option. For example, if you have 2 hosts and 3 workers, in total 6 workers are started (3 for each host).


### `loadbalance` [_loadbalance_2]

When `loadbalance: true` is set, Packetbeat connects to all configured hosts and sends data through all connections in parallel. If a connection fails, data is sent to the remaining hosts until it can be reestablished. Data will still be sent as long as Packetbeat can connect to at least one of its configured hosts.

When `loadbalance: false` is set, Packetbeat sends data to a single host at a time. The target host is chosen at random from the list of configured hosts, and all data is sent to that target until the connection fails, when a new target is selected. Data will still be sent as long as Packetbeat can connect to at least one of its configured hosts.

The default value is `true`.


### `timeout` [_timeout_5]

The Redis connection timeout in seconds. The default is 5 seconds.


### `backoff.init` [_backoff_init_3]

The number of seconds to wait before trying to reconnect to Redis after a network error. After waiting `backoff.init` seconds, Packetbeat tries to reconnect. If the attempt fails, the backoff timer is increased exponentially up to `backoff.max`. After a successful connection, the backoff timer is reset. The default is 1s.


### `backoff.max` [_backoff_max_3]

The maximum number of seconds to wait before attempting to connect to Redis after a network error. The default is 60s.


### `max_retries` [_max_retries_4]

The number of times to retry publishing an event after a publishing failure. After the specified number of retries, the events are typically dropped.

Set `max_retries` to a value less than 0 to retry until all events are published.

The default is 3.


### `bulk_max_size` [_bulk_max_size_3]

The maximum number of events to bulk in a single Redis request or pipeline. The default is 2048.

Events can be collected into batches. Packetbeat will split batches read from the queue which are larger than `bulk_max_size` into multiple batches.

Specifying a larger batch size can improve performance by lowering the overhead of sending events. However big batch sizes can also increase processing times, which might result in API errors, killed connections, timed-out publishing requests, and, ultimately, lower throughput.

Setting `bulk_max_size` to values less than or equal to 0 disables the splitting of batches. When splitting is disabled, the queue decides on the number of events to be contained in a batch.


### `ssl` [_ssl_4]

Configuration options for SSL parameters like the root CA for Redis connections guarded by SSL proxies (for example [stunnel](https://www.stunnel.org)). See [SSL](/reference/packetbeat/configuration-ssl.md) for more information.


### `proxy_url` [_proxy_url_3]

The URL of the SOCKS5 proxy to use when connecting to the Redis servers. The value must be a URL with a scheme of `socks5://`. You cannot use a web proxy because the protocol used to communicate with Redis is not based on HTTP.

If the SOCKS5 proxy server requires client authentication, you can embed a username and password in the URL.

When using a proxy, hostnames are resolved on the proxy server instead of on the client. You can change this behavior by setting the [`proxy_use_local_resolver`](#redis-proxy-use-local-resolver) option.


### `proxy_use_local_resolver` [redis-proxy-use-local-resolver]

This option determines whether Redis hostnames are resolved locally when using a proxy. The default value is false, which means that name resolution occurs on the proxy server.


### `queue` [_queue_4]

Configuration options for internal queue.

See [Internal queue](/reference/packetbeat/configuring-internal-queue.md) for more information.

Note:`queue` options can be set under `packetbeat.yml` or the `output` section but not both.



