:::{warning}
If an {{ech}} deployment is protected by an AWS PrivateLink VPC filter, you can't use `cloud.id`. The Cloud ID encodes the public {{es}} endpoint, which is not available after the filter is associated. Copy the private {{es}} endpoint from the deployment overview page, then connect using the {{es}} endpoint URL and an API key:

```yaml
output.elasticsearch:
  hosts: ["ELASTICSEARCH_ENDPOINT_URL"]
  api_key: "YOUR_API_KEY"
```
:::
