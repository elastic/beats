---
navigation_title: "Elasticsearch"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/elasticsearch-output.html
---

# Configure the Elasticsearch output [elasticsearch-output]


The Elasticsearch output sends events directly to Elasticsearch using the Elasticsearch HTTP API.

Example configuration:

```yaml
output.elasticsearch:
  hosts: ["https://myEShost:9200"] <1>
```

1. To enable SSL, add `https` to all URLs defined under *hosts*.


When sending data to a secured cluster through the `elasticsearch` output, Metricbeat can use any of the following authentication methods:

* Basic authentication credentials (username and password).
* Token-based (API key) authentication.
* Public Key Infrastructure (PKI) certificates.

**Basic authentication:**

```yaml
output.elasticsearch:
  hosts: ["https://myEShost:9200"]
  username: "metricbeat_writer"
  password: "YOUR_PASSWORD"
```

**API key authentication:**

```yaml
output.elasticsearch:
  hosts: ["https://myEShost:9200"]
  api_key: "ZCV7VnwBgnX0T19fN8Qe:KnR6yE41RrSowb0kQ0HWoA"
```

**PKI certificate authentication:**

```yaml
output.elasticsearch:
  hosts: ["https://myEShost:9200"]
  ssl.certificate: "/etc/pki/client/cert.pem"
  ssl.key: "/etc/pki/client/cert.key"
```

See [*Secure communication with Elasticsearch*](/reference/metricbeat/securing-communication-elasticsearch.md) for details on each authentication method.

## Compatibility [_compatibility]

This output works with all compatible versions of Elasticsearch. See the [Elastic Support Matrix](https://www.elastic.co/support/matrix#matrix_compatibility).

Optionally, you can set Metricbeat to only connect to instances that are at least on the same version as the Beat. The check can be enabled by setting `output.elasticsearch.allow_older_versions` to `false`. Leaving the setting at it’s default value of `true` avoids an issue where Metricbeat cannot connect to {{es}} after having been upgraded to a version higher than the {{stack}}.


## Configuration options [_configuration_options_2]

You can specify the following options in the `elasticsearch` section of the `metricbeat.yml` config file:

### `enabled` [_enabled_2]

The enabled config is a boolean setting to enable or disable the output. If set to `false`, the output is disabled.

The default value is `true`.


### `hosts` [hosts-option]

The list of Elasticsearch nodes to connect to. The events are distributed to these nodes in round robin order. If one node becomes unreachable, the event is automatically sent to another node. Each Elasticsearch node can be defined as a `URL` or `IP:PORT`. For example: `http://192.15.3.2`, `https://es.found.io:9230` or `192.24.3.2:9300`. If no port is specified, `9200` is used.

::::{note}
When a node is defined as an `IP:PORT`, the *scheme* and *path* are taken from the [`protocol`](#protocol-option) and [`path`](#path-option) config options.
::::


```yaml
output.elasticsearch:
  hosts: ["10.45.3.2:9220", "10.45.3.1:9230"] <1>
  protocol: https
  path: /elasticsearch
```

1. In the previous example, the Elasticsearch nodes are available at `https://10.45.3.2:9220/elasticsearch` and `https://10.45.3.1:9230/elasticsearch`.


### `compression_level` [compression-level-option]

The gzip compression level. Setting this value to `0` disables compression. The compression level must be in the range of `1` (best speed) to `9` (best compression).

Increasing the compression level will reduce the network usage but will increase the cpu usage.

The default value is `1`.


### `escape_html` [_escape_html]

Configure escaping of HTML in strings. Set to `true` to enable escaping.

The default value is `false`.


### `worker` or `workers` [worker-option]

The number of workers per configured host publishing events to Elasticsearch. This is best used with load balancing mode enabled. Example: If you have 2 hosts and 3 workers, in total 6 workers are started (3 for each host).

The default value is `1`.


### `loadbalance` [_loadbalance]

When `loadbalance: true` is set, Metricbeat connects to all configured hosts and sends data through all connections in parallel. If a connection fails, data is sent to the remaining hosts until it can be reestablished. Data will still be sent as long as Metricbeat can connect to at least one of its configured hosts.

When `loadbalance: false` is set, Metricbeat sends data to a single host at a time. The target host is chosen at random from the list of configured hosts, and all data is sent to that target until the connection fails, when a new target is selected. Data will still be sent as long as Metricbeat can connect to at least one of its configured hosts.

The default value is `true`.

```yaml
output.elasticsearch:
  hosts: ["localhost:9200", "localhost:9201"]
  loadbalance: true
```


### `api_key` [_api_key]

Instead of using a username and password, you can use API keys to secure communication with {{es}}. The value must be the ID of the API key and the API key joined by a colon: `id:api_key`.

See [*Grant access using API keys*](/reference/metricbeat/beats-api-keys.md) for more information.


### `username` [_username_2]

The basic authentication username for connecting to Elasticsearch.

This user needs the privileges required to publish events to {{es}}. To create a user like this, see [Create a *publishing* user](/reference/metricbeat/privileges-to-publish-events.md).


### `password` [_password_2]

The basic authentication password for connecting to Elasticsearch.


### `parameters` [_parameters]

Dictionary of HTTP parameters to pass within the url with index operations.


### `protocol` [protocol-option]

The name of the protocol Elasticsearch is reachable on. The options are: `http` or `https`. The default is `http`. However, if you specify a URL for [`hosts`](#hosts-option), the value of `protocol` is overridden by whatever scheme you specify in the URL.


### `path` [path-option]

An HTTP path prefix that is prepended to the HTTP API calls. This is useful for the cases where Elasticsearch listens behind an HTTP reverse proxy that exports the API under a custom prefix.


### `headers` [_headers_2]

Custom HTTP headers to add to each request created by the Elasticsearch output. Example:

```yaml
output.elasticsearch.headers:
  X-My-Header: Header contents
```

It is possible to specify multiple header values for the same header name by separating them with a comma.


### `proxy_disable` [_proxy_disable]

If set to `true` all proxy settings, including `HTTP_PROXY` and `HTTPS_PROXY` variables are ignored.


### `proxy_url` [_proxy_url]

The URL of the proxy to use when connecting to the Elasticsearch servers. The value must be a complete URL. If a value is not specified through the configuration file then proxy environment variables are used. See the [Go documentation](https://golang.org/pkg/net/http/#ProxyFromEnvironment) for more information about the environment variables.


### `proxy_headers` [_proxy_headers]

Additional headers to send to proxies during CONNECT requests.


### `index` [index-option-es]

The indexing target to write events to. Can point to an [index](https://www.elastic.co/guide/en/elasticsearch/reference/current/index-mgmt.html), [alias](docs-content://manage-data/data-store/aliases.md), or [data stream](docs-content://manage-data/data-store/data-streams.md). When using daily indices, this will be the index name. The default is `"metricbeat-%{[agent.version]}-%{+yyyy.MM.dd}"`, for example, `"metricbeat-[version]-2025-01-30"`. If you change this setting, you also need to configure the `setup.template.name` and `setup.template.pattern` options (see [Elasticsearch index template](/reference/metricbeat/configuration-template.md)).

If you are using the pre-built Kibana dashboards, you also need to set the `setup.dashboards.index` option (see [Kibana dashboards](/reference/metricbeat/configuration-dashboards.md)).

When [index lifecycle management (ILM)](/reference/metricbeat/ilm.md) is enabled, the default `index` is `"metricbeat-%{[agent.version]}-%{+yyyy.MM.dd}-%{{index_num}}"`, for example, `"metricbeat-[version]-2025-01-30-000001"`. Custom `index` settings are ignored when ILM is enabled. If you’re sending events to a cluster that supports index lifecycle management, see [Index lifecycle management (ILM)](/reference/metricbeat/ilm.md) to learn how to change the index name.

You can set the index dynamically by using a format string to access any event field. For example, this configuration uses a custom field, `fields.log_type`, to set the index:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  index: "%{[fields.log_type]}-%{[agent.version]}-%{+yyyy.MM.dd}" <1>
```

1. We recommend including `agent.version` in the name to avoid mapping issues when you upgrade.


With this configuration, all events with `log_type: normal` are sent to an index named `normal-[version]-2025-01-30`, and all events with `log_type: critical` are sent to an index named `critical-[version]-2025-01-30`.

::::{tip}
To learn how to add custom fields to events, see the [`fields`](/reference/metricbeat/configuration-general-options.md#libbeat-configuration-fields) option.
::::


See the [`indices`](#indices-option-es) setting for other ways to set the index dynamically.


### `indices` [indices-option-es]

An array of index selector rules. Each rule specifies the index to use for events that match the rule. During publishing, Metricbeat uses the first matching rule in the array. Rules can contain conditionals, format string-based fields, and name mappings. If the `indices` setting is missing or no rule matches, the [`index`](#index-option-es) setting is used.

Similar to `index`, defining custom `indices` will disable [Index lifecycle management (ILM)](/reference/metricbeat/ilm.md).

Rule settings:

**`index`**
:   The index format string to use. If this string contains field references, such as `%{[fields.name]}`, the fields must exist, or the rule fails.

**`mappings`**
:   A dictionary that takes the value returned by `index` and maps it to a new name.

**`default`**
:   The default string value to use if `mappings` does not find a match.

**`when`**
:   A condition that must succeed in order to execute the current rule. All the [conditions](/reference/metricbeat/defining-processors.md#conditions) supported by processors are also supported here.

The following example sets the index based on whether the `message` field contains the specified string:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  indices:
    - index: "warning-%{[agent.version]}-%{+yyyy.MM.dd}"
      when.contains:
        message: "WARN"
    - index: "error-%{[agent.version]}-%{+yyyy.MM.dd}"
      when.contains:
        message: "ERR"
```

This configuration results in indices named `warning-[version]-2025-01-30` and `error-[version]-2025-01-30` (plus the default index if no matches are found).

The following example sets the index by taking the name returned by the `index` format string and mapping it to a new name that’s used for the index:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  indices:
    - index: "%{[fields.log_type]}"
      mappings:
        critical: "sev1"
        normal: "sev2"
      default: "sev3"
```

This configuration results in indices named `sev1`, `sev2`, and `sev3`.

The `mappings` setting simplifies the configuration, but is limited to string values. You cannot specify format strings within the mapping pairs.


### `ilm` [ilm-es]

Configuration options for index lifecycle management.

See [Index lifecycle management (ILM)](/reference/metricbeat/ilm.md) for more information.


### `pipeline` [pipeline-option-es]

A format string value that specifies the ingest pipeline to write events to.

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  pipeline: my_pipeline_id
```

::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::


For more information, see [*Parse data using an ingest pipeline*](/reference/metricbeat/configuring-ingest-node.md).

You can set the ingest pipeline dynamically by using a format string to access any event field. For example, this configuration uses a custom field, `fields.log_type`, to set the pipeline for each event:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  pipeline: "%{[fields.log_type]}_pipeline"
```

With this configuration, all events with `log_type: normal` are sent to a pipeline named `normal_pipeline`, and all events with `log_type: critical` are sent to a pipeline named `critical_pipeline`.

::::{tip}
To learn how to add custom fields to events, see the [`fields`](/reference/metricbeat/configuration-general-options.md#libbeat-configuration-fields) option.
::::


See the [`pipelines`](#pipelines-option-es) setting for other ways to set the ingest pipeline dynamically.


### `pipelines` [pipelines-option-es]

An array of pipeline selector rules. Each rule specifies the ingest pipeline to use for events that match the rule. During publishing, Metricbeat uses the first matching rule in the array. Rules can contain conditionals, format string-based fields, and name mappings. If the `pipelines` setting is missing or no rule matches, the [`pipeline`](#pipeline-option-es) setting is used.

Rule settings:

**`pipeline`**
:   The pipeline format string to use. If this string contains field references, such as `%{[fields.name]}`, the fields must exist, or the rule fails.

**`mappings`**
:   A dictionary that takes the value returned by `pipeline` and maps it to a new name.

**`default`**
:   The default string value to use if `mappings` does not find a match.

**`when`**
:   A condition that must succeed in order to execute the current rule. All the [conditions](/reference/metricbeat/defining-processors.md#conditions) supported by processors are also supported here.

The following example sends events to a specific pipeline based on whether the `message` field contains the specified string:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  pipelines:
    - pipeline: "warning_pipeline"
      when.contains:
        message: "WARN"
    - pipeline: "error_pipeline"
      when.contains:
        message: "ERR"
```

The following example sets the pipeline by taking the name returned by the `pipeline` format string and mapping it to a new name that’s used for the pipeline:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  pipelines:
    - pipeline: "%{[fields.log_type]}"
      mappings:
        critical: "sev1_pipeline"
        normal: "sev2_pipeline"
      default: "sev3_pipeline"
```

With this configuration, all events with `log_type: critical` are sent to `sev1_pipeline`, all events with `log_type: normal` are sent to a `sev2_pipeline`, and all other events are sent to `sev3_pipeline`.

For more information about ingest pipelines, see [*Parse data using an ingest pipeline*](/reference/metricbeat/configuring-ingest-node.md).


### `max_retries` [_max_retries]

The number of times to retry publishing an event after a publishing failure. After the specified number of retries, the events are typically dropped.

Set `max_retries` to a value less than 0 to retry until all events are published.

The default is 3.


### `bulk_max_size` [bulk-max-size-option]

The maximum number of events to bulk in a single Elasticsearch bulk API index request. The default is 1600.

Events can be collected into batches. Metricbeat will split batches read from the queue which are larger than `bulk_max_size` into multiple batches.

Specifying a larger batch size can improve performance by lowering the overhead of sending events. However big batch sizes can also increase processing times, which might result in API errors, killed connections, timed-out publishing requests, and, ultimately, lower throughput.

Setting `bulk_max_size` to values less than or equal to 0 disables the splitting of batches. When splitting is disabled, the queue decides on the number of events to be contained in a batch.


### `backoff.init` [backoff-init-option]

The number of seconds to wait before trying to reconnect to Elasticsearch after a network error. After waiting `backoff.init` seconds, Metricbeat tries to reconnect. If the attempt fails, the backoff timer is increased exponentially up to `backoff.max`. After a successful connection, the backoff timer is reset. The default is `1s`.


### `backoff.max` [backoff-max-option]

The maximum number of seconds to wait before attempting to connect to Elasticsearch after a network error. The default is `60s`.


### `idle_connection_timeout` [idle-connection-timeout-option]

The maximum amount of time an idle connection will remain idle before closing itself. Zero means no limit. The format is a Go language duration (example 60s is 60 seconds). The default is 3s.


### `timeout` [_timeout_2]

The http request timeout in seconds for the Elasticsearch request. The default is 90.


### `allow_older_versions` [_allow_older_versions]

By default, Metricbeat expects the Elasticsearch instance to be on the same or newer version to provide optimal experience. We suggest you connect to the same version to make sure all features Metricbeat is using are available in your Elasticsearch instance.

You can disable the check for example during updating the Elastic Stack, so data collection can go on.


### `ssl` [_ssl_2]

Configuration options for SSL parameters like the certificate authority to use for HTTPS-based connections. If the `ssl` section is missing, the host CAs are used for HTTPS connections to Elasticsearch.

See the [secure communication with {{es}}](/reference/metricbeat/securing-communication-elasticsearch.md) guide or [SSL configuration reference](/reference/metricbeat/configuration-ssl.md) for more information.


### `kerberos` [_kerberos]

Configuration options for Kerberos authentication.

See [Kerberos](/reference/metricbeat/configuration-kerberos.md) for more information.


### `queue` [_queue]

Configuration options for internal queue.

See [Internal queue](/reference/metricbeat/configuring-internal-queue.md) for more information.

Note:`queue` options can be set under `metricbeat.yml` or the `output` section but not both. ===== `non_indexable_policy`

Specifies the behavior when the elasticsearch cluster explicitly rejects documents, for example on mapping conflicts.

#### `drop` [_drop]

The default behaviour, when an event is explicitly rejected by elasticsearch it is dropped.

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  non_indexable_policy.drop: ~
```


#### `dead_letter_index` [_dead_letter_index]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


On an explicit rejection, this policy will retry the event in the next batch. However, the target index will change to index specified. In addition, the structure of the event will be change to the following fields:

message
:   Contains the escaped json of the original event.

error.type
:   Contains the status code

error.message
:   Contains status returned by elasticsearch, describing the reason

`index`
:   The index to send rejected events to.

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  non_indexable_policy.dead_letter_index:
    index: "my-dead-letter-index"
```



### `preset` [_preset]

The performance preset to apply to the output configuration.

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  preset: balanced
```

Performance presets apply a set of configuration overrides based on a desired performance goal. If set, a performance preset will override other configuration flags to match the recommended settings for that preset. If a preset doesn’t set a value for a particular field, the user-specified value will be used if present, otherwise the default. Valid options are: * `balanced`: good starting point for general efficiency * `throughput`: good for high data volumes, may increase cpu and memory requirements * `scale`: reduces ambient resource use in large low-throughput deployments * `latency`: minimize the time for fresh data to become visible in Elasticsearch * `custom`: apply user configuration directly with no overrides

The default if unspecified is `custom`.

Presets represent current recommendations based on the intended goal; their effect may change between versions to better suit those goals. Currently the presets have the following effects:

| preset | balanced | throughput | scale | latency |
| --- | --- | --- | --- | --- |
| [`bulk_max_size`](#bulk-max-size-option) | 1600 | 1600 | 1600 | 50 |
| [`worker`](#worker-option) | 1 | 4 | 1 | 1 |
| [`queue.mem.events`](/reference/metricbeat/configuring-internal-queue.md#queue-mem-events-option) | 3200 | 12800 | 3200 | 4100 |
| [`queue.mem.flush.min_events`](/reference/metricbeat/configuring-internal-queue.md#queue-mem-flush-min-events-option) | 1600 | 1600 | 1600 | 2050 |
| [`queue.mem.flush.timeout`](/reference/metricbeat/configuring-internal-queue.md#queue-mem-flush-timeout-option) | `10s` | `5s` | `20s` | `1s` |
| [`compression_level`](#compression-level-option) | 1 | 1 | 1 | 1 |
| [`idle_connection_timeout`](#idle-connection-timeout-option) | `3s` | `15s` | `1s` | `60s` |
| [`backoff.init`](#backoff-init-option) | none | none | `5s` | none |
| [`backoff.max`](#backoff-max-option) | none | none | `300s` | none |



## Elasticsearch APIs [es-apis]

Metricbeat will use the `_bulk` API from {{es}}, the events are sent in the order they arrive to the publishing pipeline, a single `_bulk` request may contain events from different inputs/modules. Temporary failures are re-tried.

The status code for each event is checked and handled as:

* `< 300`: The event is counted as `events.acked`
* `409` (Conflict): The event is counted as `events.duplicates`
* `429` (Too Many Requests): The event is counted as `events.toomany`
* `> 399 and < 500`: The `non_indexable_policy` is applied.


