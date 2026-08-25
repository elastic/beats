:::{warning}
If you connect over AWS PrivateLink, you can't use `cloud.id`. The Cloud ID encodes the public {{es}} endpoint, which doesn't match the PrivateLink TLS certificate. Copy the private {{es}} endpoint from the deployment overview page, then connect using the {{es}} endpoint URL and an API key:

```yaml
output.elasticsearch:
  hosts: ["ELASTICSEARCH_ENDPOINT_URL"]
  api_key: "YOUR_API_KEY"
```
:::
