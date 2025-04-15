---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-monitor.html
---

# Settings for internal collection [configuration-monitor]

Use the following settings to configure internal collection when you are not using {{metricbeat}} to collect monitoring data.

You specify these settings in the X-Pack monitoring section of the `metricbeat.yml` config file:

## `monitoring.enabled` [_monitoring_enabled]

The `monitoring.enabled` config is a boolean setting to enable or disable {{monitoring}}. If set to `true`, monitoring is enabled.

The default value is `false`.


## `monitoring.elasticsearch` [_monitoring_elasticsearch]

The {{es}} instances that you want to ship your Metricbeat metrics to. This configuration option contains the following fields:


## `monitoring.cluster_uuid` [_monitoring_cluster_uuid]

The `monitoring.cluster_uuid` config identifies the {{es}} cluster under which the monitoring data will appear in the Stack Monitoring UI.

### `api_key` [_api_key_3]

The detail of the API key to be used to send monitoring information to {{es}}. See [*Grant access using API keys*](/reference/metricbeat/beats-api-keys.md) for more information.


### `bulk_max_size` [_bulk_max_size_5]

The maximum number of metrics to bulk in a single {{es}} bulk API index request. The default is `50`. For more information, see [Elasticsearch](/reference/metricbeat/elasticsearch-output.md).


### `backoff.init` [_backoff_init_4]

The number of seconds to wait before trying to reconnect to Elasticsearch after a network error. After waiting `backoff.init` seconds, Metricbeat tries to reconnect. If the attempt fails, the backoff timer is increased exponentially up to `backoff.max`. After a successful connection, the backoff timer is reset. The default is 1s.


### `backoff.max` [_backoff_max_4]

The maximum number of seconds to wait before attempting to connect to Elasticsearch after a network error. The default is 60s.


### `compression_level` [_compression_level_3]

The gzip compression level. Setting this value to `0` disables compression. The compression level must be in the range of `1` (best speed) to `9` (best compression). The default value is `0`. Increasing the compression level reduces the network usage but increases the CPU usage.


### `headers` [_headers_4]

Custom HTTP headers to add to each request. For more information, see [Elasticsearch](/reference/metricbeat/elasticsearch-output.md).


### `hosts` [_hosts_5]

The list of {{es}} nodes to connect to. Monitoring metrics are distributed to these nodes in round robin order. For more information, see [Elasticsearch](/reference/metricbeat/elasticsearch-output.md).


### `max_retries` [_max_retries_5]

The number of times to retry sending the monitoring metrics after a failure. After the specified number of retries, the metrics are typically dropped. The default value is `3`. For more information, see [Elasticsearch](/reference/metricbeat/elasticsearch-output.md).


### `parameters` [_parameters_2]

Dictionary of HTTP parameters to pass within the url with index operations.


### `password` [_password_6]

The password that Metricbeat uses to authenticate with the {{es}} instances for shipping monitoring data.


### `metrics.period` [_metrics_period]

The time interval (in seconds) when metrics are sent to the {{es}} cluster. A new snapshot of Metricbeat metrics is generated and scheduled for publishing each period. The default value is 10 * time.Second.


### `state.period` [_state_period]

The time interval (in seconds) when state information are sent to the {{es}} cluster. A new snapshot of Metricbeat state is generated and scheduled for publishing each period. The default value is 60 * time.Second.


### `protocol` [_protocol]

The name of the protocol to use when connecting to the {{es}} cluster. The options are: `http` or `https`. The default is `http`. If you specify a URL for `hosts`, however, the value of protocol is overridden by the scheme you specify in the URL.


### `proxy_url` [_proxy_url_4]

The URL of the proxy to use when connecting to the {{es}} cluster. For more information, see [Elasticsearch](/reference/metricbeat/elasticsearch-output.md).


### `timeout` [_timeout_6]

The HTTP request timeout in seconds for the {{es}} request. The default is `90`.


### `ssl` [_ssl_9]

Configuration options for Transport Layer Security (TLS) or Secure Sockets Layer (SSL) parameters like the certificate authority (CA) to use for HTTPS-based connections. If the `ssl` section is missing, the host CAs are used for HTTPS connections to {{es}}. For more information, see [SSL](/reference/metricbeat/configuration-ssl.md).


### `username` [_username_5]

The user ID that Metricbeat uses to authenticate with the {{es}} instances for shipping monitoring data.



