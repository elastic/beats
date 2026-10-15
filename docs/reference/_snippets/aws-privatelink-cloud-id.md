:::{warning}
If an {{ech}} deployment is protected by an AWS PrivateLink VPC filter, you can't use `cloud.id`. The Cloud ID encodes the public {{es}} endpoint, which is not available after the filter is associated. [Find the private {{es}} URL](docs-content://deploy-manage/security/private-connectivity-aws.md#ec-access-the-deployment-over-private-link), then connect using that URL and an API key:

```yaml
output.elasticsearch:
  hosts: ["ELASTICSEARCH_ENDPOINT_URL"]
  api_key: "YOUR_API_KEY"
```
:::
