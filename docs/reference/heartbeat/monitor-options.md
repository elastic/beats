---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/monitor-options.html
applies_to:
  stack: ga
  serverless: ga
---

# Common monitor options [monitor-options]

You can specify the following options when defining a Heartbeat monitor in any location. These options are the same for all monitors. Each monitor type has additional configuration options that are specific to that monitor type.


### `type` [monitor-type]

The type of monitor to run. See [Monitor types](/reference/heartbeat/configuration-heartbeat-options.md#monitor-types).


### `id` [monitor-id]

A unique identifier for this configuration. This should not change with edits to the monitor configuration regardless of changes to any config fields. Examples: `uploader-service`, `http://example.net`, `us-west-loadbalancer`. Note that this uniqueness is only within a given beat instance. If you want to monitor the same endpoint from multiple locations it is recommended that those heartbeat instances use the same IDs so that their results can be correlated. You can use the `host.geo.name` property to disambiguate them.

When querying against indexed monitor data this is the field you will be aggregating with. Appears in the [exported fields](/reference/heartbeat/exported-fields.md) as `monitor.id`.

If you do not set this explicitly the monitor’s config will be hashed and a generated value used. This value will change with any options change to this monitor making aggregations over time between changes impossible. For this reason it is recommended that you set this manually.


### `name` [monitor-name]

Optional human readable name for this monitor. This value appears in the [exported fields](/reference/heartbeat/exported-fields.md) as `monitor.name`.


### `service.name` [service-name]

Optional APM service name for this monitor. Corresponds to the `service.name` ECS field. Set this when monitoring an app that is also using APM to enable integrations between Uptime and APM data in Kibana.


### `enabled` [monitor-enabled]

A Boolean value that specifies whether the module is enabled. If the `enabled` option is missing from the configuration block, the module is enabled by default.


### `schedule` [monitor-schedule]

A cron-like expression that specifies the task schedule. For example:

* `*/5 * * * * * *` runs the task every 5 seconds (for example, at 10:00:00, 10:00:05, and so on).
* `@every 5s` runs the task every 5 seconds from the time when Heartbeat was started.

The `schedule` option uses a cron-like syntax based on [this `cronexpr` implementation](https://github.com/gorhill/cronexpr#implementation), but adds the `@every` keyword.

For stats on the execution of scheduled tasks you can enable the HTTP stats server with `http.enabled: true` in heartbeat.yml, then run `curl http://localhost:5066/stats | jq .heartbeat.scheduler` to view the scheduler’s stats. Stats are provided for both jobs and tasks. Each time a monitor is scheduled is considered to be a single job, while portions of the work a job does, like DNS lookups and executing network requests are defined as tasks. The stats provided are:

* **jobs.active:** The number of actively running jobs/monitors.
* **jobs.missed_deadline:** The number of jobs that executed after their scheduled time. This can be caused either by overlong long timeouts from the previous job or high load preventing heartbeat from keeping up with work.
* **tasks.active:** The number of tasks currently running.
* **tasks.waiting:** If the global `schedule.limit` option is set, this number will reflect the number of tasks that are ready to execute, but have not been started in order to prevent exceeding `schedule.limit`.

Also see the [task scheduler](/reference/heartbeat/monitors-scheduler.md) settings.


### `ipv4` [monitor-ipv4]

A Boolean value that specifies whether to ping using the ipv4 protocol if hostnames are configured. The default is `true`.


### `ipv6` [monitor-ipv6]

A Boolean value that specifies whether to ping using the ipv6 protocol if hostnames are configured. The default is `true`.


### `mode` [monitor-mode]

If `mode` is `any`, the monitor pings only one IP address for a hostname. If `mode` is `all`, the monitor pings all resolvable IPs for a hostname. The `mode: all` setting is useful if you are using a DNS-load balancer and want to ping every IP address for the specified hostname. The default is `any`.


### `timeout` [monitor-timeout]

The total running time for each ping test. This is the total time allowed for testing the connection and exchanging data. The default is 16 seconds (16s).

If the timeout is exceeded, Heartbeat publishes a `service-down` event. If the value specified for `timeout` is greater than `schedule`, intermediate checks will not be executed by the scheduler.


## `run_from` [monitor-run-from]

Use the `run_from` option to set the geographic location fields relevant to a given heartbeat monitor.

The `run_from` option is used to label the geographic location where the monitor is running. Note, you can also set the `run_from` option on all monitors via the `heartbeat.run_from` option.

The `run_from` option takes two top-level fields:

* `id`: A string used to uniquely identify the geographic location. It is indexed as the `observer.name` field.
* `geo`: A map conforming to [ECS geo fields](ecs://reference/ecs-geo.md). It is indexed under `observer.geo`.

Example:

```yaml
- type: http
  # Set enabled to true (or delete the following line) to enable this example monitor
  enabled: true
  # ID used to uniquely identify this monitor in elasticsearch even if the config changes
  id: geo-test
  # Human readable display name for this service in Uptime UI and elsewhere
  name: Geo Test
  # List or urls to query
  urls: ["http://example.net"]
  # Configure task schedule
  schedule: '@every 10s'
  run_from:
    id: my-custom-geo
    geo:
      name: nyc-dc1-rack1
      location: 40.7128, -74.0060
      continent_name: North America
      country_iso_code: US
      region_name: New York
      region_iso_code: NY
      city_name: New York
```


## `secret_stores` [monitor-secret-stores]

```{applies_to}
stack: preview 9.6+
```

Use the `secret_stores` option to read secrets from an external secret store when the monitor runs, instead of writing them in the monitor configuration. Heartbeat supports HashiCorp Vault KV v2 secrets.

Reference a secret as `vault://<name>@<path>#<field>`, where `<name>` is the `name` of the `secret_stores` entry holding the connection, in any string setting of the monitor. A reference can be part of a longer value, such as `"Bearer vault://prod@my-app/api#token"`. The path is relative to `kv_mount` and can't contain `.` or `..` segments. Percent-encode `#` and `@` in the path or field, as `%23` and `%40`. A reference can also be the value of a [keystore](/reference/heartbeat/keystore.md) or environment variable.

The name selects the connection, so references can use several Vault connections. Heartbeat reports an error for a reference without a name or with a name no entry has. Percent-encode characters of the name that are not allowed in a reference, such as `@`.

Example:

```yaml
- type: http
  id: my-api
  schedule: '@every 1m'
  urls: ["https://api.example.com/status"]
  check.request.headers:
    Authorization: "Bearer vault://prod@my-app/api#token"
  secret_stores:
    - type: vault
      name: prod
      address: https://vault.example.com:8200
      auth:
        approle:
          role_id: my-role-id
          secret_id: my-secret-id
```

Each `secret_stores` entry takes these options:

`type`
:   The secret store type. Must be `vault`.

`name`
:   The name that references use to select this entry, as in `vault://prod@my-app/api#token`. Names must be unique.

`address`
:   The URL of the Vault server.

`namespace`
:   (Optional) The Vault Enterprise namespace.

`kv_mount`
:   (Optional) The mount path of the KV v2 secrets engine. Defaults to `secret`.

`auth`
:   The authentication method, exactly one of:

    * `token`: A Vault token.
    * `approle`: AppRole credentials: `role_id`, `secret_id`, and optionally `mount_path`, which defaults to `approle`.

`ssl`
:   (Optional) TLS settings of the connection to Vault, such as `certificate_authorities` to trust a private CA. They only apply to Vault: the `ssl` settings of the monitor apply to the monitored host. See [SSL](/reference/heartbeat/configuration-ssl.md) for the available settings.

`allowed_hosts`
:   (Optional) The hosts that the secrets of this connection can be sent to, as host names or IP addresses, without scheme, port, or path. See [Restrict the hosts that receive secrets](#monitor-secret-stores-allowed-hosts). If you don't set it, or set it to an empty list, any host can receive the secrets.

Heartbeat resolves the references before each run. A secret is read from Vault at most once every 5 minutes and shared by the monitors that use the same connection, so rotated secrets are used within 5 minutes without changing the monitor. On runs that read from Vault, the monitor duration includes the time of the read.

If a reference can't be resolved, for example because Vault is unavailable or the path doesn't exist, the monitor reports `down` with the error and tries again on its next run. Failed reads are not retried for 30 seconds.

References in the `source.inline.script` setting of browser monitors are not resolved. Pass secrets to scripts with `params` instead.

If a monitor with several URLs or hosts starts while Vault is unavailable, or if the resolved values change its number of URLs or hosts, it reports an error until you change its configuration or restart Heartbeat.

::::{warning}
Heartbeat uses the resolved values as they are. A secret in a setting that is published in events, such as a URL or a host, is visible in those events, for example in `url.full`, `url.query`, or `error.message`, as with any other secret in the configuration. Use settings that are not published for secrets, such as `username`, `password`, request headers, or TLS settings.
::::

### Restrict the hosts that receive secrets [monitor-secret-stores-allowed-hosts]

```{applies_to}
stack: preview 9.6+
```

Use `allowed_hosts` to make sure that the secrets of a connection are only sent to the hosts you approve, whoever writes the monitors that use it. This holds only if the monitor authors can't change the `secret_stores` setting, for example when Kibana adds it to the monitors. If the same person writes the monitor and its `secret_stores`, as in a standalone `heartbeat.yml`, they can remove `allowed_hosts` or add their own connection. Before each run, Heartbeat resolves the references and then checks the resolved monitor configuration. A monitor that fails the check doesn't run and reports `down` with an error such as:

```
secret_stores: connection "prod": not allowed by allowed_hosts: the monitor connects to a host that is not allowed
```

The error doesn't name the host, because a host can come from a secret.

A monitor that lists the connection in `secret_stores` but doesn't reference it is not checked.

A monitor that references the connection passes the check when:

* Its type is `http`, `tcp`, or `icmp`. Browser monitors can't use the connection, because their scripts can send the secrets to any host.
* Every host it connects to is in `allowed_hosts`: the hosts of `urls` or `hosts`, and the host of `proxy_url` if it's set.
* It's an `http` monitor with `max_redirects` set to `0`, the default, or a monitor of another type. A redirect can send the request to any host.
* It's an `http` monitor without `kerberos`, or a monitor of another type. Kerberos sends data derived from the password to the Kerberos servers (KDCs), which are not checked.

Host names are compared without case and without a trailing dot. Ports are ignored, so `api.example.com` allows `https://api.example.com:8443`. IP addresses are compared as addresses, so `::1` allows `[0:0:0:0:0:0:0:1]`. Internationalized host names are compared in the ASCII form that Heartbeat connects to, so `bücher.example` allows `xn--bcher-kva.example`. Wildcards are not supported.

Heartbeat checks the hosts after it resolves the references, so the hosts that are checked are the ones the monitor connects to. For example, `https://attacker.example/?k=vault://prod@payments/stripe#api_key` is rejected because `attacker.example` is not allowed, and so is a URL or a host that comes from a secret and isn't in `allowed_hosts`.

Because the check runs after the references are resolved, a monitor that fails it still reads its secrets from Vault on its runs, at most once every 5 minutes for each secret. These reads appear in the Vault audit log. The secrets stay in Heartbeat and are not sent to any host.

Example:

```yaml
- type: http
  id: stripe-charges
  schedule: '@every 1m'
  urls: ["https://api.stripe.com/v1/charges"]
  check.request.headers:
    Authorization: "Bearer vault://prod@payments/stripe#api_key"
  secret_stores:
    - type: vault
      name: prod
      address: https://vault.example.com:8200
      auth:
        token: my-token
      allowed_hosts: ["api.stripe.com"]
```

`allowed_hosts` restricts where the secrets are sent, not who can see them. A secret in a setting that Heartbeat publishes is visible in the events, as the warning above describes.


### `fields` [monitor-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.


### `fields_under_root` [monitor-fields-under-root]

If this option is set to true, the custom [fields](#monitor-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Heartbeat, then the custom fields overwrite the other fields.


### `tags` [monitor-tags]

A list of tags that will be sent with the monitor event. This setting is optional.


### `processors` [monitor-processors]

A list of processors to apply to the data generated by the monitor.

See [Processors](/reference/heartbeat/filtering-enhancing-data.md) for information about specifying processors in your config.


### `data_stream` [monitor-data-stream]

Contains options pertaining to data stream naming, following the conventions followed by [Fleet Data Streams](docs-content://reference/fleet/data-streams.md). By default Heartbeat will write to a datastream named `heartbeat-VERSION`.

```yaml
# To enable data streams with the default namespace
data_stream.namespace: default
```


#### `pipeline` [monitor-pipeline]

The {{es}} ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::



#### `index` (deprecated) [monitor-index]

This setting is now deprecated in favor of the `data_stream` option. If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"heartbeat-myindex-2019.11.01"`.


### `keep_null` [monitor-keep-null]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.
