Set the {{es}} endpoint and API key in your Beat configuration file. To find your project's endpoint and create an API key, refer to [connection details](docs-content://solutions/search/search-connection-details.md). For example:

```yaml
output.elasticsearch:
  hosts: ["ELASTICSEARCH_ENDPOINT_URL"]
  api_key: "YOUR_API_KEY" <1>
```

1. This example shows a hard-coded API key, but you should store sensitive values in the secrets keystore. Refer to [Grant access using API keys](/reference/filebeat/beats-api-keys.md) for more on API key configuration.

::::{note}
Do not use `cloud.id` or `cloud.auth` for {{es-serverless}} projects. Those settings are for {{ech}} deployments only.
::::

