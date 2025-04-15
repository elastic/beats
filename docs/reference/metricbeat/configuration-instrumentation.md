---
navigation_title: "Instrumentation"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-instrumentation.html
---

# Configure APM instrumentation [configuration-instrumentation]


Libbeat uses the Elastic APM Go Agent to instrument its publishing pipeline. Currently, only the Elasticsearch output is instrumented. To gain insight into the performance of Metricbeat, you can enable this instrumentation and send trace data to the APM Integration.

Example configuration with instrumentation enabled:

```yaml
instrumentation:
  enabled: true
  environment: production
  hosts:
    - "http://localhost:8200"
  api_key: L5ER6FEvjkmlfalBealQ3f3fLqf03fazfOV
```


## Configuration options [_configuration_options_16]

You can specify the following options in the `instrumentation` section of the `metricbeat.yml` config file:


### `enabled` [_enabled_10]

Set to `true` to enable instrumentation of Metricbeat. Defaults to `false`.


### `environment` [_environment]

Set the environment in which Metricbeat is running, for example, `staging`, `production`, `dev`, etc. Environments can be filtered in the [APM app](docs-content://solutions/observability/apm/overviews.md).


### `hosts` [_hosts_4]

The APM integration [host](docs-content://reference/apm/observability/apm-settings.md) to report instrumentation data to. Defaults to `http://localhost:8200`.


### `api_key` [_api_key_2]

The [API Key](docs-content://reference/apm/observability/apm-settings.md) used to secure communication with the APM Integration. If `api_key` is set then `secret_token` will be ignored.


### `secret_token` [_secret_token]

The [Secret token](docs-content://reference/apm/observability/apm-settings.md) used to secure communication with the APM Integration.


### `profiling.cpu.enabled` [_profiling_cpu_enabled]

Set to `true` to enable CPU profiling, where profile samples are recorded as events.

This feature is experimental.


### `profiling.cpu.interval` [_profiling_cpu_interval]

Configure the CPU profiling interval. Defaults to `60s`.

This feature is experimental.


### `profiling.cpu.duration` [_profiling_cpu_duration]

Configure the CPU profiling duration. Defaults to `10s`.

This feature is experimental.


### `profiling.heap.enabled` [_profiling_heap_enabled]

Set to `true` to enable heap profiling.

This feature is experimental.


### `profiling.heap.interval` [_profiling_heap_interval]

Configure the heap profiling interval. Defaults to `60s`.

This feature is experimental.

