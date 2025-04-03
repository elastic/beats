---
navigation_title: "Logstash"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/logstash-output.html
---

# Configure the Logstash output [logstash-output]


The {{ls}} output sends events directly to {{ls}} by using the lumberjack protocol, which runs over TCP. {{ls}} allows for additional processing and routing of generated events.

::::{admonition} Prerequisite
:class: important

To send events to {{ls}}, you also need to create a {{ls}} configuration pipeline that listens for incoming Beats connections and indexes the received events into {{es}}. For more information, see [Getting Started with {{ls}}](logstash://reference/getting-started-with-logstash.md). Also see the documentation for the [{{beats}} input](logstash-docs-md://lsr/plugins-inputs-beats.md) and [{{es}} output](logstash-docs-md://lsr/plugins-outputs-elasticsearch.md) plugins.
::::


If you want to use {{ls}} to perform additional processing on the data collected by Filebeat, you need to configure Filebeat to use {{ls}}.

To do this, edit the Filebeat configuration file to disable the {{es}} output by commenting it out and enable the {{ls}} output by uncommenting the {{ls}} section:

```yaml
output.logstash:
  hosts: ["127.0.0.1:5044"]
```

The `hosts` option specifies the {{ls}} server and the port (`5044`) where {{ls}} is configured to listen for incoming Beats connections.

For this configuration, you must [load the index template into {{es}} manually](/reference/filebeat/filebeat-template.md#load-template-manually) because the options for auto loading the template are only available for the {{es}} output.

Want to use [Filebeat modules](/reference/filebeat/filebeat-modules.md) with {{ls}}? You need to do some extra setup. For more information, see [Working with Filebeat modules](logstash://reference/working-with-filebeat-modules.md).

## Accessing metadata fields [_accessing_metadata_fields]

Every event sent to {{ls}} contains the following metadata fields that you can use in {{ls}} for indexing and filtering:

```json subs=true
{
    ...
    "@metadata": { <1>
      "beat": "filebeat", <2>
      "version": "{{stack-version}}" <3>
    }
}
```

1. Filebeat uses the `@metadata` field to send metadata to {{ls}}. See the [{{ls}} documentation](logstash://reference/event-dependent-configuration.md#metadata) for more about the `@metadata` field.
2. The default is filebeat. To change this value, set the [`index`](#logstash-index) option in the Filebeat config file.
3. The current version of Filebeat.


You can access this metadata from within the {{ls}} config file to set values dynamically based on the contents of the metadata.

For example, the following {{ls}} configuration file tells {{ls}} to use the index reported by Filebeat for indexing events into {{es}}:

```json
input {
  beats {
    port => 5044
  }
}

output {
  elasticsearch {
    hosts => ["http://localhost:9200"]
    index => "%{[@metadata][beat]}-%{[@metadata][version]}" <1>
    action => "create"
  }
}
```

1. `%{[@metadata][beat]}` sets the first part of the index name to the value of the `beat` metadata field and `%{[@metadata][version]}` sets the second part to the Beat’s version. For example: `filebeat-9[version]`.


Events indexed into {{es}} with the {{ls}} configuration shown here will be similar to events directly indexed by Filebeat into {{es}}.

::::{note}
If ILM is not being used, set `index` to `%{[@metadata][beat]}-%{[@metadata][version]}-%{+YYYY.MM.dd}` instead so {{ls}} creates an index per day, based on the `@timestamp` value of the events coming from Beats.
::::



## Compatibility [_compatibility_2]

This output works with all compatible versions of {{ls}}. See the [Elastic Support Matrix](https://www.elastic.co/support/matrix#matrix_compatibility).


## Configuration options [_configuration_options_26]

You can specify the following options in the `logstash` section of the `filebeat.yml` config file:

### `enabled` [_enabled_31]

The enabled config is a boolean setting to enable or disable the output. If set to false, the output is disabled.

The default value is `true`.


### `hosts` [hosts]

The list of known {{ls}} servers to connect to. If load balancing is disabled, but multiple hosts are configured, one host is selected randomly (there is no precedence). If one host becomes unreachable, another one is selected randomly.

All entries in this list can contain a port number. The default port number 5044 will be used if no number is given.


### `compression_level` [_compression_level]

The gzip compression level. Setting this value to 0 disables compression. The compression level must be in the range of 1 (best speed) to 9 (best compression).

Increasing the compression level will reduce the network usage but will increase the CPU usage.

The default value is 3.


### `escape_html` [_escape_html_2]

Configure escaping of HTML in strings. Set to `true` to enable escaping.

The default value is `false`.


### `worker` or `workers` [_worker_or_workers]

The number of workers per configured host publishing events to {{ls}}. This is best used with load balancing mode enabled. Example: If you have 2 hosts and 3 workers, in total 6 workers are started (3 for each host).


### `loadbalance` [loadbalance]

When `loadbalance: true` is set, Filebeat connects to all configured hosts and sends data through all connections in parallel. If a connection fails, data is sent to the remaining hosts until it can be reestablished. Data will still be sent as long as Filebeat can connect to at least one of its configured hosts.

When `loadbalance: false` is set, Filebeat sends data to a single host at a time. The target host is chosen at random from the list of configured hosts, and all data is sent to that target until the connection fails, when a new target is selected. Data will still be sent as long as Filebeat can connect to at least one of its configured hosts. To rotate through the list of configured hosts over time, use this option in conjunction with the `ttl` setting to close the connection at the configured interval and choose a new target host.

The default value is `false`.

```yaml
output.logstash:
  hosts: ["localhost:5044", "localhost:5045"]
  loadbalance: true
  index: filebeat
```


### `ttl` [_ttl]

Time to live for a connection to {{ls}} after which the connection will be re-established. Useful when {{ls}} hosts represent load balancers. Since the connections to {{ls}} hosts are sticky, operating behind load balancers can lead to uneven load distribution between the instances. Specifying a TTL on the connection allows to achieve equal connection distribution between the instances.  Specifying a TTL of 0 will disable this feature.

The default value is 0. This setting accepts [duration](/reference/libbeat/config-file-format-type.md#_duration) data type values.

::::{note}
The "ttl" option is not yet supported on an async {{ls}} client (one with the "pipelining" option set).
::::



### `pipelining` [_pipelining]

Configures the number of batches to be sent asynchronously to {{ls}} while waiting for ACK from {{ls}}. Output only becomes blocking once number of `pipelining` batches have been written. Pipelining is disabled if a value of 0 is configured. The default value is 2.


### `proxy_url` [_proxy_url_3]

The URL of the SOCKS5 proxy to use when connecting to the {{ls}} servers. The value must be a URL with a scheme of `socks5://`. The protocol used to communicate to {{ls}} is not based on HTTP so a web-proxy cannot be used.

If the SOCKS5 proxy server requires client authentication, then a username and password can be embedded in the URL as shown in the example.

When using a proxy, hostnames are resolved on the proxy server instead of on the client. You can change this behavior by setting the [`proxy_use_local_resolver`](#logstash-proxy-use-local-resolver) option.

```yaml
output.logstash:
  hosts: ["remote-host:5044"]
  proxy_url: socks5://user:password@socks5-proxy:2233
```


### `proxy_use_local_resolver` [logstash-proxy-use-local-resolver]

The `proxy_use_local_resolver` option determines if {{ls}} hostnames are resolved locally when using a proxy. The default value is false, which means that when a proxy is used the name resolution occurs on the proxy server.


### `index` [logstash-index]

The index root name to write events to. The default is the Beat name. For example `"filebeat"` generates `"[filebeat-][version]-YYYY.MM.DD"` indices (for example, `"filebeat-9.0.0-2017.04.26"`).

::::{note}
This parameter’s value will be assigned to the `metadata.beat` field. It can then be accessed in {{ls}}'s output section as `%{[@metadata][beat]}`.
::::



### `ssl` [_ssl_5]

Configuration options for SSL parameters like the root CA for {{ls}} connections. See [SSL](/reference/filebeat/configuration-ssl.md) for more information. To use SSL, you must also configure the [Beats input plugin for Logstash](logstash-docs-md://lsr/plugins-inputs-beats.md) to use SSL/TLS.


### `timeout` [_timeout_3]

The number of seconds to wait for responses from the {{ls}} server before timing out. The default is 30 (seconds).


### `max_retries` [_max_retries_2]

Filebeat ignores the `max_retries` setting and retries indefinitely.


### `bulk_max_size` [_bulk_max_size]

The maximum number of events to bulk in a single {{ls}} request. The default is 2048.

Events can be collected into batches. Filebeat will split batches read from the queue which are larger than `bulk_max_size` into multiple batches.

Specifying a larger batch size can improve performance by lowering the overhead of sending events. However big batch sizes can also increase processing times, which might result in API errors, killed connections, timed-out publishing requests, and, ultimately, lower throughput.

Setting `bulk_max_size` to values less than or equal to 0 disables the splitting of batches. When splitting is disabled, the queue decides on the number of events to be contained in a batch.


### `slow_start` [_slow_start]

If enabled, only a subset of events in a batch of events is transferred per transaction. The number of events to be sent increases up to `bulk_max_size` if no error is encountered. On error, the number of events per transaction is reduced again.

The default is `false`.


### `backoff.init` [_backoff_init_2]

The number of seconds to wait before trying to reconnect to {{ls}} after a network error. After waiting `backoff.init` seconds, Filebeat tries to reconnect. If the attempt fails, the backoff timer is increased exponentially up to `backoff.max`. After a successful connection, the backoff timer is reset. The default is 1s.


### `backoff.max` [_backoff_max_2]

The maximum number of seconds to wait before attempting to connect to {{ls}} after a network error. The default is 60s.


### `queue` [_queue_2]

Configuration options for internal queue.

See [Internal queue](/reference/filebeat/configuring-internal-queue.md) for more information.

Note:`queue` options can be set under `filebeat.yml` or the `output` section but not both.



