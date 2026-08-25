---
navigation_title: "{{ech}}"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/configure-cloud-id.html
applies_to:
  stack: ga
  serverless: ga
---

# Configure the output for {{ech}} [configure-cloud-id]


Auditbeat comes with two settings that simplify the output configuration when used together with [{{ech}}](https://www.elastic.co/cloud?page=docs&placement=docs-body). When defined, these setting overwrite settings from other parts in the configuration.

Example:

```yaml
cloud.id: "staging:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw=="
cloud.auth: "elastic:YOUR_PASSWORD"
```

These settings can be also specified at the command line, like this:

```sh
auditbeat -e -E cloud.id="<cloud-id>" -E cloud.auth="<cloud.auth>"
```

## `cloud.id` [_cloud_id]

The Cloud ID, which can be found in the [{{ecloud}} console](https://cloud.elastic.co/?page=docs&placement=docs-body), is used by Auditbeat to resolve the {{es}} and {{kib}} URLs. This setting overwrites the `output.elasticsearch.hosts` and `setup.kibana.host` settings. For more on locating and configuring the Cloud ID, see [Find your Cloud ID](docs-content://deploy-manage/deploy/elastic-cloud/find-cloud-id.md).

:::{warning}
If you connect over AWS PrivateLink, don't set `cloud.id`. The Cloud ID uses the public endpoint, which doesn't match the PrivateLink TLS certificate. Use the {{es}} endpoint URL instead.
:::

Set `output.elasticsearch.hosts` to the private URL for your deployment. Don't also set `cloud.id`. That setting overwrites `output.elasticsearch.hosts`.

```yaml
output.elasticsearch:
  hosts: ["https://my-deployment-d53192.es.vpce.us-east-1.aws.elastic-cloud.com:443"]
  username: "elastic"
  password: "YOUR_PASSWORD"
```

For the private URL, refer to [Access the resource over a PrivateLink](docs-content://deploy-manage/security/private-connectivity-aws.md#ec-access-the-deployment-over-private-link).

## `cloud.auth` [_cloud_auth]

::::{important}
`cloud.auth` should not be confused with API keys generated in {{ecloud}} stack management. Although these values look similar, they are unrelated.
::::

When specified, the `cloud.auth` overwrites the `output.elasticsearch.username` and `output.elasticsearch.password` settings. Because the Kibana settings inherit the username and password from the {{es}} output, this can also be used to set the `setup.kibana.username` and `setup.kibana.password` options.
