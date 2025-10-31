---
navigation_title: "Azure Event Hub"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-azure-eventhub.html
applies_to:
  stack: ga
---

# Azure eventhub input [filebeat-input-azure-eventhub]


Users can make use of the `azure-eventhub` input in order to read messages from an azure eventhub. The azure-eventhub input implementation is based on the the event processor host (EPH is intended to be run across multiple processes and machines while load balancing message consumers more on this here [https://github.com/Azure/azure-event-hubs-go#event-processor-host](https://github.com/Azure/azure-event-hubs-go#event-processor-host), [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host)). State such as leases on partitions and checkpoints in the event stream are shared between receivers using an Azure Storage container. For this reason, as a prerequisite to using this input, users will have to create or use an existing storage account.

Users can enable internal logs tracing for this input by setting the environment variable `BEATS_AZURE_EVENTHUB_INPUT_TRACING_ENABLED: true`. When enabled, this input will log additional information to the logs. Additional information includes partition ownership, blob lease information, and other internal state.

Example configuration using Shared Access Key authentication:

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "test"
  connection_string: "Endpoint=sb://....."
  storage_account: "azureeph"
  storage_account_key: "....."
  storage_account_container: ""
  resource_manager_endpoint: ""
```

{applies_to}`stack: ga 9.3.0` Example configuration using OAuth2 authentication:

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "test"
  # No connection_string provided - automatically uses OAuth2 for both eventhub and storage account
  eventhub_namespace: "your-eventhub-namespace.servicebus.windows.net"
  tenant_id: "your-tenant-id"
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  authority_host: "https://login.microsoftonline.com"
  storage_account: "azureeph"
  storage_account_container: ""
  processor_version: "v2"
```

## Configuration options [_configuration_options]

The `azure-eventhub` input supports the following configuration:


## `eventhub` [_eventhub]

The name of the eventhub users would like to read from, field required.


## `consumer_group` [_consumer_group]

Optional, we recommend using a dedicated consumer group for the azure input. Reusing consumer groups among non-related consumers can cause unexpected behavior and possibly lost events.


## `connection_string` [_connection_string]

The connection string required to communicate with Event Hubs when using Shared Access Key authentication, steps here [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string).

**Note**: If `connection_string` is not provided, the input will automatically use OAuth2 authentication and require the OAuth2 configuration parameters below.

A Blob Storage account is required in order to store/retrieve/update the offset or state of the eventhub messages. This means that after stopping filebeat it can start back up at the spot that it stopped processing messages.

## `eventhub_namespace` [_eventhub_namespace]

```{applies_to}
stack: ga 9.3.0
```

The fully qualified namespace for the Event Hub. Required when `connection_string` is not provided (OAuth2 authentication). Format: `your-eventhub-namespace.servicebus.windows.net`

## `tenant_id` [_tenant_id]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory tenant ID. Required when `connection_string` is not provided (OAuth2 authentication).

## `client_id` [_client_id]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory application (client) ID. Required when `connection_string` is not provided (OAuth2 authentication).

## `client_secret` [_client_secret]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory application client secret. Required when `connection_string` is not provided (OAuth2 authentication).

## `authority_host` [_authority_host]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory authority host. Optional when using OAuth2 authentication. Defaults to Azure Public Cloud (`https://login.microsoftonline.com`).

Supported values:
- `https://login.microsoftonline.com` (Azure Public Cloud - default)
- `https://login.microsoftonline.us` (Azure Government)
- `https://login.chinacloudapi.cn` (Azure China)


## `storage_account` [_storage_account]

The name of the storage account. Required.


## `storage_account_key` [_storage_account_key]

The storage account key, this key will be used to authorize access to data in your storage account, option is required.


## `storage_account_container` [_storage_account_container]

Optional, the name of the storage account container you would like to store the offset information in.


## `resource_manager_endpoint` [_resource_manager_endpoint]

Optional, by default we are using the azure public environment, to override, users can provide a specific resource manager endpoint in order to use a different azure environment. Ex: [https://management.chinacloudapi.cn/](https://management.chinacloudapi.cn/) for azure ChinaCloud [https://management.microsoftazure.de/](https://management.microsoftazure.de/) for azure GermanCloud [https://management.azure.com/](https://management.azure.com/) for azure PublicCloud [https://management.usgovcloudapi.net/](https://management.usgovcloudapi.net/) for azure USGovernmentCloud Users can also use this in case of a Hybrid Cloud model, where one may define their own endpoints.


## Metrics [_metrics_3]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `received_messages_total` | Number of messages received from the event hub. |
| `received_bytes_total` | Number of bytes received from the event hub. |
| `sanitized_messages_total` | Number of messages that were sanitized successfully. |
| `processed_messages_total` | Number of messages that were processed successfully. |
| `received_events_total` | Number of events received decoding messages. |
| `sent_events_total` | Number of events that were sent successfully. |
| `processing_time` | Histogram of the elapsed processing times in nanoseconds. |
| `decode_errors_total` | Number of errors that occurred while decoding a message. |


