# Elasticsearch authentication extension

`elasticsearchauth` is a Development OTel `extensionauth.HTTPClient` authenticator
for Elasticsearch consumers. It owns only resolved Elasticsearch endpoints,
destination headers, and Basic/API-key credentials.

Simple mode uses the consumer-supplied base transport directly:

```yaml
extensions:
  elasticsearchauth/default:
    endpoints: [https://es.example:9200]
    user: elastic
    password: ${env:ELASTICSEARCH_PASSWORD}
    headers:
      X-Elastic-Product-Origin: custom
```

Delegated mode configures `beatsauth` when transport-specific settings are
needed:

```yaml
extensions:
  beatsauth/default:
    ssl:
      certificate_authorities: [/path/to/ca.pem]
    proxy_url: https://proxy.example:8443

  elasticsearchauth/default:
    endpoints: [https://es.example:9200]
    api_key: <id>:<key>
    headers:
      X-Elastic-Product-Origin: custom
    auth:
      authenticator: beatsauth/default
```

Only plural `endpoints` is accepted. Each must be a fully resolved HTTP(S) URL.
`api_key` is the raw, non-empty Beats-style `id:key`; the extension base64-encodes
it exactly once when constructing `Authorization: ApiKey <base64(id:key)>`.
Alternatively configure both `user` and `password`. Endpoint userinfo cannot be
combined with explicit credentials or an `Authorization` destination header.

When `auth.authenticator` performs HTTP authentication, do not also configure
`user`, `password`, `api_key`, or an `Authorization` header on
`elasticsearchauth`. The outer wrapper applies its credentials before handing
the request to the nested authenticator; both authenticators can otherwise set
the `Authorization` header. This cannot be validated from the
`elasticsearchauth` configuration because nested authenticator settings are not
available there.

Warning: `beatsauth` with Kerberos enabled applies SPNEGO authentication. Do
not combine it with any `elasticsearchauth` credentials or `Authorization`
header.

Without `auth`, `RoundTripper` wraps the consumer-supplied base transport
directly. With `auth`, it passes that base transport to the configured
authenticator and wraps the returned transport. In both modes, the outer
wrapper applies Elasticsearch credentials and destination headers, and forwards
`CloseIdleConnections` when the wrapped transport supports it. Configure TLS,
certificate behavior, proxies, dialing, keepalive, Kerberos, and every other
transport concern on `beatsauth`; transport fields on `elasticsearchauth` are
rejected by strict configuration decoding.

## Elasticsearch storage configuration

`elasticsearch_storage` uses an `elasticsearchauth` extension for both its
destination endpoints and its HTTP transport:

```yaml
extensions:
  elasticsearchauth/state:
    endpoints: [https://es.example:9200]
    api_key: <base64-encoded-id:key>

  elasticsearch_storage:
    auth:
      authenticator: elasticsearchauth/state
```

Configure destination and authentication settings on `elasticsearchauth`;
`elasticsearch_storage` no longer accepts independent Elasticsearch connection
settings.

## Elasticsearch output compatibility

| Output setting | Extension behavior or required configuration |
| --- | --- |
| `hosts`, `protocol`, `path`, query parameters, Cloud ID, env substitutions | Configure their fully resolved results as plural `endpoints`; the extension does not perform endpoint resolution. |
| `username`/`password`, `api_key` | Configure either `user` and `password`, or a raw, non-empty Beats-style `id:key` as `api_key`; the mechanisms are mutually exclusive, and the extension owns base64 encoding for the authorization header. |
| `headers` | Configure `headers` directly on `elasticsearchauth`; an `Authorization` header cannot be combined with Basic/API-key credentials. |
| TLS, certificate reload, `ca_trusted_fingerprint`, verification modes | Configure on an optional `beatsauth` extension and reference it through `auth.authenticator`. |
| `proxy_url`, `proxy_headers`, `proxy_disable` | Configure on an optional `beatsauth` extension and reference it through `auth.authenticator`. |
| dialing, connection pool, keepalive, Kerberos | Configure on an optional `beatsauth` extension and reference it through `auth.authenticator`. |
| nested HTTP client authentication | Omit `auth` for simple mode, or configure `auth.authenticator` with the component ID of a transport-owning `beatsauth` extension. |
