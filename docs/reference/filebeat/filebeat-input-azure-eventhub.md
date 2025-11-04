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

{applies_to}`stack: ga 9.3.0` Example configuration using client secret authentication:

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "test"
  auth_type: "client_secret"
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


## Authentication [_authentication]

The azure-eventhub input supports multiple authentication methods. The `auth_type` configuration option controls the authentication method used for both Event Hub and Storage Account.

### Authentication Types

The following authentication types are supported:

- **`connection_string`** (default if `auth_type` is not specified): Uses Azure Event Hubs and Storage Account connection strings
- **`client_secret`**: Uses Azure Active Directory service principal with client secret credentials

### Required Permissions

When using `client_secret` authentication, the service principal needs the following Azure RBAC permissions:

**For Azure Event Hubs:**
- `Azure Event Hubs Data Receiver` role on the Event Hubs namespace or Event Hub
- Alternatively, a custom role with the following permissions:
  - `Microsoft.EventHub/namespaces/eventhubs/read`
  - `Microsoft.EventHub/namespaces/eventhubs/consumergroups/read`

**For Azure Storage Account:**
- `Storage Blob Data Contributor` role on the Storage Account or container
- Alternatively, a custom role with the following permissions:
  - `Microsoft.Storage/storageAccounts/blobServices/containers/read`
  - `Microsoft.Storage/storageAccounts/blobServices/containers/write`
  - `Microsoft.Storage/storageAccounts/blobServices/containers/delete`
  - `Microsoft.Storage/storageAccounts/blobServices/generateUserDelegationKey/action`

For detailed instructions on how to set up an Azure AD service principal and configure permissions, refer to the official Microsoft documentation:

- [Create an Azure service principal with Azure CLI](https://learn.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli)
- [Create an Azure AD app registration using the Azure portal](https://learn.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal)
- [Assign Azure roles using Azure CLI](https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-cli)
- [Azure Event Hubs authentication and authorization](https://learn.microsoft.com/en-us/azure/event-hubs/authorize-access-azure-active-directory)
- [Authorize access to blobs using Azure Active Directory](https://learn.microsoft.com/en-us/azure/storage/blobs/authorize-access-azure-active-directory)

## `connection_string` [_connection_string]

The connection string required to communicate with Event Hubs when using `connection_string` authentication, steps here [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string).

Required when `auth_type` is set to `connection_string` or when `auth_type` is not specified (defaults to `connection_string` for backwards compatibility).

A Blob Storage account is required in order to store/retrieve/update the offset or state of the eventhub messages. This means that after stopping filebeat it can start back up at the spot that it stopped processing messages.

## `auth_type` [_auth_type]

```{applies_to}
stack: ga 9.3.0
```

Specifies the authentication method to use for both Event Hub and Storage Account. If not specified, defaults to `connection_string` for backwards compatibility.

Valid values:
- `connection_string`: Uses connection string authentication (default)
- `client_secret`: Uses Azure Active Directory service principal with client secret credentials

## `eventhub_namespace` [_eventhub_namespace]

```{applies_to}
stack: ga 9.3.0
```

The fully qualified namespace for the Event Hub. Required when using `client_secret` authentication (`auth_type` is set to `client_secret`). Format: `your-eventhub-namespace.servicebus.windows.net`

## `tenant_id` [_tenant_id]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory tenant ID. Required when using `client_secret` authentication for Event Hub or Storage Account.

## `client_id` [_client_id]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory application (client) ID. Required when using `client_secret` authentication for Event Hub or Storage Account.

## `client_secret` [_client_secret]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory application client secret. Required when using `client_secret` authentication for Event Hub or Storage Account.

## `authority_host` [_authority_host]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory authority host. Optional when using `client_secret` authentication. Defaults to Azure Public Cloud (`https://login.microsoftonline.com`).

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


