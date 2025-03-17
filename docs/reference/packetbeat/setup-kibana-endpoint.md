---
navigation_title: "{{kib}} endpoint"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/setup-kibana-endpoint.html
---

# Configure the {{kib}} endpoint [setup-kibana-endpoint]


{{kib}} dashboards are loaded into {{kib}} via the {{kib}} API. This requires a {{kib}} endpoint configuration. For details on authenticating to the {{kib}} API, see [Authentication](https://www.elastic.co/docs/api/doc/kibana/authentication).

You configure the endpoint in the `setup.kibana` section of the `packetbeat.yml` config file.

Here is an example configuration:

```yaml
setup.kibana.host: "http://localhost:5601"
```


## Configuration options [_configuration_options_25]

You can specify the following options in the `setup.kibana` section of the `packetbeat.yml` config file:


### `setup.kibana.host` [_setup_kibana_host]

The {{kib}} host where the dashboards will be loaded. The default is `127.0.0.1:5601`. The value of `host` can be a `URL` or `IP:PORT`. For example: `http://192.15.3.2`, `192:15.3.2:5601` or `http://192.15.3.2:6701/path`. If no port is specified, `5601` is used.

::::{note}
When a node is defined as an `IP:PORT`, the *scheme* and *path* are taken from the [setup.kibana.protocol](#kibana-protocol-option) and [setup.kibana.path](#kibana-path-option) config options.
::::


IPv6 addresses must be defined using the following format: `https://[2001:db8::1]:5601`.


### `setup.kibana.protocol` [kibana-protocol-option]

The name of the protocol {{kib}} is reachable on. The options are: `http` or `https`. The default is `http`. However, if you specify a URL for host, the value of `protocol` is overridden by whatever scheme you specify in the URL.

Example config:

```yaml
setup.kibana.host: "192.0.2.255:5601"
setup.kibana.protocol: "http"
setup.kibana.path: /kibana
```


### `setup.kibana.username` [_setup_kibana_username]

The basic authentication username for connecting to {{kib}}. If you don’t specify a value for this setting, Packetbeat uses the `username` specified for the {{es}} output.


### `setup.kibana.password` [_setup_kibana_password]

The basic authentication password for connecting to {{kib}}. If you don’t specify a value for this setting, Packetbeat uses the `password` specified for the {{es}} output.


### `setup.kibana.path` [kibana-path-option]

An HTTP path prefix that is prepended to the HTTP API calls. This is useful for the cases where {{kib}} listens behind an HTTP reverse proxy that exports the API under a custom prefix.


### `setup.kibana.space.id` [kibana-space-id-option]

The [Kibana space](docs-content://deploy-manage/manage-spaces.md) ID to use. If specified, Packetbeat loads {{kib}} assets into this {{kib}} space. Omit this option to use the default space.


#### `setup.kibana.headers` [_setup_kibana_headers]

Custom HTTP headers to add to each request sent to {{kib}}. Example:

```yaml
setup.kibana.headers:
  X-My-Header: Header contents
```


### `setup.kibana.ssl.enabled` [_setup_kibana_ssl_enabled]

Enables Packetbeat to use SSL settings when connecting to {{kib}} via HTTPS. If you configure Packetbeat to connect over HTTPS, this setting defaults to `true` and Packetbeat uses the default SSL settings.

Example configuration:

```yaml
setup.kibana.host: "https://192.0.2.255:5601"
setup.kibana.ssl.enabled: true
setup.kibana.ssl.certificate_authorities: ["/etc/client/ca.pem"]
setup.kibana.ssl.certificate: "/etc/client/cert.pem"
setup.kibana.ssl.key: "/etc/client/cert.key
```

See [SSL](/reference/packetbeat/configuration-ssl.md) for more information.

