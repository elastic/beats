---
navigation_title: "Azure Event Hub"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-azure-eventhub.html
---

# Azure eventhub input [filebeat-input-azure-eventhub]


Users can make use of the `azure-eventhub` input in order to read messages from an azure eventhub. The azure-eventhub input implementation is based on the the event processor host (EPH is intended to be run across multiple processes and machines while load balancing message consumers more on this here [https://github.com/Azure/azure-event-hubs-go#event-processor-host](https://github.com/Azure/azure-event-hubs-go#event-processor-host), [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host)). State such as leases on partitions and checkpoints in the event stream are shared between receivers using an Azure Storage container. For this reason, as a prerequisite to using this input, users will have to create or use an existing storage account.

Users can enable internal logs tracing for this input by setting the environment variable `BEATS_AZURE_EVENTHUB_INPUT_TRACING_ENABLED: true`. When enabled, this input will log additional information to the logs. Additional information includes partition ownership, blob lease information, and other internal state.

Example configuration:

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

## Configuration options [_configuration_options]

The `azure-eventhub` input supports the following configuration:


## `eventhub` [_eventhub]

The name of the eventhub users would like to read from, field required.


## `consumer_group` [_consumer_group]

Optional, we recommend using a dedicated consumer group for the azure input. Reusing consumer groups among non-related consumers can cause unexpected behavior and possibly lost events.


## `connection_string` [_connection_string]

The connection string required to communicate with Event Hubs, steps here [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string).

A Blob Storage account is required in order to store/retrieve/update the offset or state of the eventhub messages. This means that after stopping filebeat it can start back up at the spot that it stopped processing messages.


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


