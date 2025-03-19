---
navigation_title: "{{ess}}"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configure-cloud-id.html
---

# Configure the output for {{ess}} on {{ecloud}} [configure-cloud-id]


Filebeat comes with two settings that simplify the output configuration when used together with [{{ess}}](https://www.elastic.co/cloud/elasticsearch-service?page=docs&placement=docs-body). When defined, these setting overwrite settings from other parts in the configuration.

Example:

```yaml
cloud.id: "staging:dXMtZWFzdC0xLmF3cy5mb3VuZC5pbyRjZWM2ZjI2MWE3NGJmMjRjZTMzYmI4ODExYjg0Mjk0ZiRjNmMyY2E2ZDA0MjI0OWFmMGNjN2Q3YTllOTYyNTc0Mw=="
cloud.auth: "elastic:{pwd}"
```

These settings can be also specified at the command line, like this:

```sh
filebeat -e -E cloud.id="<cloud-id>" -E cloud.auth="<cloud.auth>"
```

## `cloud.id` [_cloud_id]

The Cloud ID, which can be found in the {{ess}} web console, is used by Filebeat to resolve the {{es}} and {{kib}} URLs. This setting overwrites the `output.elasticsearch.hosts` and `setup.kibana.host` settings. For more on locating and configuring the Cloud ID, see [Configure Beats and Logstash with Cloud ID](docs-content://deploy-manage/deploy/cloud-enterprise/find-cloud-id.md).


## `cloud.auth` [_cloud_auth]

When specified, the `cloud.auth` overwrites the `output.elasticsearch.username` and `output.elasticsearch.password` settings. Because the Kibana settings inherit the username and password from the {{es}} output, this can also be used to set the `setup.kibana.username` and `setup.kibana.password` options.


