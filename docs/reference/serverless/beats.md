---
mapped_pages:
  - https://www.elastic.co/guide/en/serverless/current/elasticsearch-ingest-data-through-beats.html
---

# Beats [elasticsearch-ingest-data-through-beats]

{{beats}} are lightweight data shippers that send operational data to {{es}}. Elastic provides separate {{beats}} for different types of data, such as logs, metrics, and uptime. Depending on what data you want to collect, you may need to install multiple shippers on a single host.

| Data | {{beats}} |
| --- | --- |
| Audit data | [Auditbeat](https://www.elastic.co/products/beats/auditbeat) |
| Log files and journals | [Filebeat](https://www.elastic.co/products/beats/filebeat) |
| Availability | [Heartbeat](https://www.elastic.co/products/beats/heartbeat) |
| Metrics | [Metricbeat](https://www.elastic.co/products/beats/metricbeat) |
| Network traffic | [Packetbeat](https://www.elastic.co/products/beats/packetbeat) |
| Windows event logs | [Winlogbeat](https://www.elastic.co/products/beats/winlogbeat) |

{{beats}} can send data to {{es}} directly or through {{ls}}, where you can further process and enhance the data before visualizing it in {{kib}}.

::::{admonition} Authenticating with {{es}}
:class: note

When you use {{beats}} to export data to an {{es-serverless}} project, the {{beats}} require an API key to authenticate with {{es}}. Refer to [Create API key](docs-content://solutions/search/search-connection-details.md#create-an-api-key-serverless) for the steps to set up your API key, and to [Grant access using API keys](https://www.elastic.co/guide/en/beats/filebeat/current/beats-api-keys.md) in the Filebeat documentation for an example of how to configure your {{beats}} to use the key.

::::


Check out [Get started with Beats](/reference/index.md) for some next steps.

