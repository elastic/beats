# Elasticsearch storage extension

`elasticsearch_storage` is a Development-stability OpenTelemetry Collector
extension used by cloud-native (agentless) integrations — such as those backed
by Filebeat `cel`, `httpjson`, and `entity-analytics` inputs — to persist input
cursors and state in Elasticsearch (`agentless-state-<input-id>` indices) rather
than on local disk.

It implements `backend.Registry` (`github.com/elastic/beats/v7/libbeat/statestore/backend`)
and `entcollect.Registry` (`github.com/elastic/entcollect`).

## Enabling in Filebeat / Elastic Agent

Filebeat only instantiates and routes input state to `elasticsearch_storage`
when the `AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES` environment variable
is set to a comma-separated list of input types (for example,
`AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES=httpjson,cel`).

In OTel receiver mode (`filebeatreceiver`), the receiver must also reference the
extension via `storage: elasticsearch_storage` (wired automatically by Elastic
Agent when `AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES` is set).

## Configuration

`elasticsearch_storage` requires an HTTP client authenticator extension under
`auth.authenticator` that implements both `extensionauth.HTTPClient` and
`EndpointsProvider` (`Endpoints() []string`).

### Preferred: `elasticsearchauth`

`elasticsearchauth` is the preferred authenticator for `elasticsearch_storage`:

```yaml
extensions:
  elasticsearchauth/state:
    endpoints:
      - https://es.example:9200
    api_key: ${env:ELASTICSEARCH_API_KEY}
  elasticsearch_storage:
    auth:
      authenticator: elasticsearchauth/state

service:
  extensions:
    - elasticsearchauth/state
    - elasticsearch_storage
```

### Legacy compatibility: `beatsauth`

`beatsauth` is also accepted by `elasticsearch_storage` to support legacy/older
deployments where policies do not yet define an `elasticsearchauth` extension:

```yaml
extensions:
  beatsauth/state:
    endpoints:
      - https://es.example:9200
    auth:
      api_key: ${env:ELASTICSEARCH_API_KEY}
  elasticsearch_storage:
    auth:
      authenticator: beatsauth/state

service:
  extensions:
    - beatsauth/state
    - elasticsearch_storage
```

## Settings

| Setting | Description |
| --- | --- |
| `auth.authenticator` | Required. Component ID of the authenticator extension (prefer `elasticsearchauth[/<name>]`; `beatsauth[/<name>]` is supported for backward compatibility) that supplies the Elasticsearch endpoints and authenticated HTTP transport. |
