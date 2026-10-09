# Beats HTTP authentication extension

`beatsauth` is a Development OpenTelemetry `extensionauth.HTTPClient`
authenticator. It creates a Beats HTTP transport for collector components and
optionally exposes resolved Elasticsearch endpoints.

The extension owns transport configuration: TLS, proxying, timeouts, connection
reuse, APM instrumentation, HTTP authentication, and Kerberos. `RoundTripper`
returns this configured transport; it does not use the consumer-provided base
transport.

## Configuration

```yaml
extensions:
  beatsauth/elasticsearch:
    endpoints:
      - https://es.example:9200
    ssl:
      certificate_authorities: [/etc/ssl/elastic-ca.pem]
    proxy_url: https://proxy.example:8443
    timeout: 60s
    idle_connection_timeout: 30s
    auth:
      username: elastic
      password: ${env:ELASTICSEARCH_PASSWORD}
      headers:
        - key: X-Elastic-Product-Origin
          value: otel
```

`endpoints` is optional. When configured, every entry must be a fully resolved
HTTP(S) URL without URL userinfo or fragments. The extension exposes a copy
through its `Endpoints()` method for consumers that need an Elasticsearch
destination. It does not resolve hosts, paths, query parameters, Cloud IDs, or
environment substitutions.

TLS, proxy, timeout, keepalive, and certificate-reload options use the existing
Beats HTTP transport configuration. Omit `auth` to create an unauthenticated
transport while retaining those transport settings.

## HTTP authentication

`auth` is optional and is provided by
`httpcommon.HTTPTransportSettings`.

| Setting | Result |
| --- | --- |
| `auth.username` and `auth.password` | Adds `Authorization: Basic <base64(username:password)>` when both are non-empty. |
| `auth.api_key` | Adds `Authorization: ApiKey <value>` without transforming the value. Supply the exact value Elasticsearch expects. |
| `auth.headers` | Adds each configured `{key, value}` header when that header is absent from the consumer request. |

Configure one authorization mechanism. In particular, do not combine
`auth.api_key`, Basic credentials, or an `Authorization` entry in
`auth.headers`; each mechanism can set `Authorization`.

```yaml
# API key
extensions:
  beatsauth/api-key:
    auth:
      api_key: ${env:ELASTICSEARCH_API_KEY}

# Custom authorization or destination headers
extensions:
  beatsauth/headers:
    auth:
      headers:
        - key: Authorization
          value: Bearer ${env:ELASTICSEARCH_TOKEN}
        - key: X-Elastic
          value: otel
```

## Kerberos and startup errors

Configure Kerberos under `kerberos`; it adds SPNEGO authentication over the
configured HTTP transport. Do not combine it with Basic authentication, API key
authentication, or an `Authorization` header.

By default, an invalid transport configuration prevents the extension from
starting. Set `continue_on_error: true` to allow startup, but all requests using
the extension then fail with the transport-creation error.
