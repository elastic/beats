---
mapped_pages:
  - https://www.elastic.co/guide/en/serverless/current/elasticsearch-ingest-data-through-beats.html
applies_to:
  stack: ga
  serverless: ga
---

# Beats for {{es-serverless}} [elasticsearch-ingest-data-through-beats]

{{beats}} are lightweight data shippers that send operational data to {{es}}. Elastic provides separate {{beats}} for different types of data, such as logs, metrics, and uptime. Depending on what data you want to collect, you might need to install multiple shippers on a single host.

{{beats}} are not hosted by Elastic. You deploy and manage them on your own infrastructure, such as on-premises servers, virtual machines, or containers. {{beats}} work with all {{es-serverless}} project types, including Elasticsearch, Observability, and Security projects.

::::{tip}
If you're looking for a hosted data collection option that doesn't require managing infrastructure, consider [agentless integrations](docs-content://solutions/security/get-started/agentless-integrations.md), which run on Elastic's infrastructure and require no agent deployment or maintenance.
::::

| Data | {{beats}} |
| --- | --- |
| Audit data | [Auditbeat](/reference/auditbeat/index.md) |
| Log files and journals | [Filebeat](/reference/filebeat/index.md) |
| Availability | [Heartbeat](/reference/heartbeat/index.md) |
| Metrics | [Metricbeat](/reference/metricbeat/index.md) |
| Network traffic | [Packetbeat](/reference/packetbeat/index.md) |
| Windows event logs | [Winlogbeat](/reference/winlogbeat/index.md) |

{{beats}} can send data to {{es}} directly or through {{ls}}, where you can further process and enhance the data before visualizing it in {{kib}}.

## Set up {{beats}} with {{es-serverless}} [serverless-beats-setup]

To send data to an {{es-serverless}} project, configure your Beat to connect using the project's {{es}} endpoint URL and an API key.

### Get your connection details [serverless-connection-details]

1. Log in to [Elastic Cloud](https://cloud.elastic.co/).
2. Find your **{{es}} endpoint URL**. Select **Manage** next to your project, then find the {{es}} endpoint under **Application endpoints, cluster and component IDs**. Alternatively, open your project, select the help icon, then select **Connection details**.
3. Create an **API key** with the appropriate privileges. Refer to [Create API key](docs-content://solutions/search/search-connection-details.md#create-an-api-key-serverless) for detailed steps. For information on the required privileges, refer to [Grant access using API keys](/reference/filebeat/beats-api-keys.md).

### Configure the output [serverless-configure-output]

In your Beat configuration file (for example, `filebeat.yml`), set the `output.elasticsearch` section with your endpoint URL and API key:

```yaml
output.elasticsearch:
  hosts: ["ELASTICSEARCH_ENDPOINT_URL"]
  api_key: "YOUR_API_KEY"
```

::::{note}
Do not use `cloud.id` or `cloud.auth` for {{es-serverless}} projects. Those settings are for [{{ech}}](/reference/filebeat/configure-cloud-id.md) deployments only.
::::

### Install and start your Beat [serverless-install-start]

Follow the quick start guide for the Beat you want to use:

- [Auditbeat quick start](/reference/auditbeat/auditbeat-installation-configuration.md)
- [Filebeat quick start](/reference/filebeat/filebeat-installation-configuration.md)
- [Heartbeat quick start](/reference/heartbeat/heartbeat-installation-configuration.md)
- [Metricbeat quick start](/reference/metricbeat/metricbeat-installation-configuration.md)
- [Packetbeat quick start](/reference/packetbeat/packetbeat-installation-configuration.md)
- [Winlogbeat quick start](/reference/winlogbeat/winlogbeat-installation-configuration.md)

When you reach the connection setup step, use the {{es-serverless}} configuration from [Configure the output](#serverless-configure-output) instead of the `cloud.id` or `hosts` examples shown for other deployment types.

## Differences from other deployment types [serverless-differences]

When using {{beats}} with {{es-serverless}}, keep the following differences in mind:

- **Authentication**: {{es-serverless}} requires API key authentication. Username and password authentication, `cloud.id`, and `cloud.auth` are not supported.
- **Data stream lifecycle**: {{es-serverless}} uses data stream lifecycle (DSL) instead of index lifecycle management (ILM). ILM settings in your Beat configuration are ignored. Refer to the [data stream lifecycle](docs-content://manage-data/lifecycle/data-stream.md) documentation for details.
- **Ingest pipelines**: Ingest pipelines work the same way as in other deployment types.

