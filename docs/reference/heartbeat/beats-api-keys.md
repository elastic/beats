---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/beats-api-keys.html
---

# Grant access using API keys [beats-api-keys]

Instead of using usernames and passwords, you can use API keys to grant access to {{es}} resources. You can set API keys to expire at a certain time, and you can explicitly invalidate them. Any user with the `manage_api_key` or `manage_own_api_key` cluster privilege can create API keys.

Heartbeat instances typically send both collected data and monitoring information to {{es}}. If you are sending both to the same cluster, you can use the same API key. For different clusters, you need to use an API key per cluster.

::::{note}
For security reasons, we recommend using a unique API key per Heartbeat instance. You can create as many API keys per user as necessary.
::::


::::{important}
Review [*Grant users access to secured resources*](/reference/heartbeat/feature-roles.md) before creating API keys for Heartbeat.
::::



## Create an API key for publishing [beats-api-key-publish]

To create an API key to use for writing data to {{es}}, use the [Create API key API](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-create-api-key), for example:

```console
POST /_security/api_key
{
  "name": "heartbeat_host001", <1>
  "role_descriptors": {
    "heartbeat_writer": { <2>
      "cluster": ["monitor", "read_ilm", "read_pipeline"],
      "index": [
        {
          "names": ["heartbeat-*"],
          "privileges": ["view_index_metadata", "create_doc", "auto_configure"]
        }
      ]
    }
  }
}
```

1. Name of the API key
2. Granted privileges, see [*Grant users access to secured resources*](/reference/heartbeat/feature-roles.md)


::::{note}
See [Create a *publishing* user](/reference/heartbeat/privileges-to-publish-events.md) for the list of privileges required to publish events.
::::


The return value will look something like this:

```console-result
{
  "id":"TiNAGG4BaaMdaH1tRfuU", <1>
  "name":"heartbeat_host001",
  "api_key":"KnR6yE41RrSowb0kQ0HWoA" <2>
}
```

1. Unique id for this API key
2. Generated API key


You can now use this API key in your `heartbeat.yml` configuration file like this:

```yaml
output.elasticsearch:
  api_key: TiNAGG4BaaMdaH1tRfuU:KnR6yE41RrSowb0kQ0HWoA <1>
```

1. Format is `id:api_key` (as returned by [Create API key](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-create-api-key))



## Create an API key for monitoring [beats-api-key-monitor]

To create an API key to use for sending monitoring data to {{es}}, use the [Create API key API](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-create-api-key), for example:

```console
POST /_security/api_key
{
  "name": "heartbeat_host001", <1>
  "role_descriptors": {
    "heartbeat_monitoring": { <2>
      "cluster": ["monitor"],
      "index": [
        {
          "names": [".monitoring-beats-*"],
          "privileges": ["create_index", "create"]
        }
      ]
    }
  }
}
```

1. Name of the API key
2. Granted privileges, see [*Grant users access to secured resources*](/reference/heartbeat/feature-roles.md)


::::{note}
See [Create a *monitoring* user](/reference/heartbeat/privileges-to-publish-monitoring.md) for the list of privileges required to send monitoring data.
::::


The return value will look something like this:

```console-result
{
  "id":"TiNAGG4BaaMdaH1tRfuU", <1>
  "name":"heartbeat_host001",
  "api_key":"KnR6yE41RrSowb0kQ0HWoA" <2>
}
```

1. Unique id for this API key
2. Generated API key


You can now use this API key in your `heartbeat.yml` configuration file like this:

```yaml
monitoring.elasticsearch:
  api_key: TiNAGG4BaaMdaH1tRfuU:KnR6yE41RrSowb0kQ0HWoA <1>
```

1. Format is `id:api_key` (as returned by [Create API key](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-create-api-key))



## Learn more about API keys [learn-more-api-keys]

See the {{es}} API key documentation for more information:

* [Create API key](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-create-api-key)
* [Get API key information](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-get-api-key)
* [Invalidate API key](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-security-invalidate-api-key)

