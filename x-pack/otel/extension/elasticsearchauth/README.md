# Elasticsearch authentication extension

`elasticsearchauth` is a Development-stability OTel `extensionauth.HTTPClient`
extension for components that talk to Elasticsearch. It provides:

- the resolved Elasticsearch `endpoints` (through the `EndpointsProvider`
  interface, `Endpoints() []string`);
- a `RoundTripper` that builds a fresh, consumer-owned HTTP transport on every
  call and injects headers and credentials into each request.

```yaml
extensions:
  elasticsearchauth/default:
    endpoints:
      - https://es1.example:9200
      - https://es2.example:9200
    api_key: <api-key>
    headers:
      X-Custom-Header: <value>
    tls:
      ca_file: /path/to/ca.pem
    proxy_url: https://proxy.example:8443
```

## Configuration

| Setting | Description |
| --- | --- |
| `endpoints` | Required. Fully resolved HTTP(S) URLs. Each needs an `http` or `https` scheme and a non-empty host, and cannot contain a fragment or userinfo. |
| `api_key` | Sent as `Authorization: ApiKey <api_key>`. The value is passed through as is; its format is not validated. |
| `user`, `password` | HTTP Basic authentication. Must be configured together. Mutually exclusive with `api_key`. |
| `headers` | Headers added to every request, for example `X-Custom-Header`. A `Host` entry (case-insensitive) overrides the request host. An `Authorization` header cannot be combined with `api_key`, `user` or `password`. |
| `tls` | Standard OTel `configtls` client settings (`ca_file`, `cert_file`, `key_file`, `insecure_skip_verify`, ...). |
| `proxy_url` | HTTP(S) proxy URL. |
| `max_conns_per_host`, `keepalive`, `read_buffer_size`, `write_buffer_size`, `force_attempt_http2`, `http2_read_idle_timeout`, `http2_ping_timeout` | Standard `confighttp` transport settings. |

`confighttp` settings that do not apply are rejected: `endpoint` (use
`endpoints`), `timeout` (consumers own request deadlines), `auth`,
`middlewares`, `cookies`, and `compression`.

## Behavior

- `Start` and `Shutdown` are no-ops.
- TLS files are loaded and validated when the extension is created. Client
  certificate and key files (`tls.cert_file`, `tls.key_file`) are reloaded on
  subsequent TLS handshakes when `tls.reload_interval` is set; CA certificates
  (`tls.ca_file`) are loaded once at creation.
- `RoundTripper` ignores the base transport it receives. TLS, proxy, pool,
  keepalive and HTTP/2 settings come from this extension, and each call returns
  an independent `*http.Transport`.
- The returned round tripper clones each request, so the caller's request is
  never modified, and implements `CloseIdleConnections`.
- Credentials and header values are `configopaque` values and are redacted when
  the configuration is printed.
