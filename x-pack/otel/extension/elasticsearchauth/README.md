# Elasticsearch authentication extension

`elasticsearchauth` is a Development OTel `extensionauth.HTTPClient` authenticator
for Elasticsearch consumers. It exposes resolved `endpoints`, creates a fresh
consumer-owned HTTP transport for every `RoundTripper` call.

```yaml
extensions:
  elasticsearchauth/default:
    endpoints: [https://es.example:9200]
    api_key: <base64-encoded-id:key>
    tls:
      ca_file: /path/to/ca.pem
    proxy_url: https://proxy.example:8443
```

Only plural `endpoints` is accepted. Each must be a fully resolved HTTP(S) URL.
`api_key` is base64-encoded `id:key`; alternatively configure both `user` and
`password`. Endpoint userinfo cannot be combined with explicit
credentials. Consumers own request deadlines.

TLS files are loaded and validated when the extension is created. Certificate
rotation after creation is not supported. TLS, proxy, headers, pool,
keepalive, and HTTP/2 settings are authoritative here and do not come
from the base transport passed to `RoundTripper`.

## Elasticsearch output compatibility

| Output setting | Extension behavior or required configuration |
| --- | --- |
| `hosts`, `protocol`, `path`, query parameters, Cloud ID, env substitutions | Configure their fully resolved results as plural `endpoints`; the extension does not perform endpoint resolution. |
| `username`/`password`, `api_key` | Configure either `user` and `password`, or a base64-encoded `api_key`; the mechanisms are mutually exclusive. |
| TLS CA/certificate/key, `insecure`, server name | Configure the equivalent OTel `tls` fields; the extension loads and validates TLS files when created. |
| `proxy_url` | Configure the OTel field of the same name. |
| `headers` | Configure OTel `headers`. |
| connection pool, keepalive, HTTP/2 values | Configure the corresponding OTel `confighttp` fields. |
| output `timeout` | Not supported by the extension; consumers own request deadlines. |
| `proxy_headers`, `proxy_disable` | Unsupported by the initial contract; remove or replace these settings before configuring the extension. |
| `ca_trusted_fingerprint`, nonstandard TLS verification modes | Unsupported by the initial contract; provide equivalent standard OTel `tls` configuration where possible. |
