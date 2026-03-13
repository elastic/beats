---
navigation_title: "Azure Event Hub"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-azure-eventhub.html
applies_to:
  stack: ga
  serverless: ga
---

# Azure eventhub input [filebeat-input-azure-eventhub]

Use the `azure-eventhub` input to read messages from an Azure EventHub. The azure-eventhub input implementation is based on the event processor host. EPH is intended to be run across multiple processes and machines while load balancing message consumers more on this here [https://github.com/Azure/azure-event-hubs-go#event-processor-host](https://github.com/Azure/azure-event-hubs-go#event-processor-host), [https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host). 

State such as leases on partitions and checkpoints in the event stream are shared between receivers using an Azure Storage container. For this reason, as a prerequisite to using this input, you must create or use an existing storage account.

Enable internal logs tracing for this input by setting the environment variable `BEATS_AZURE_EVENTHUB_INPUT_TRACING_ENABLED: true`. When enabled, this input will log additional information to the logs. Additional information includes partition ownership, blob lease information, and other internal state.

## Processor versions [_processor_versions]

The `azure-eventhub` input supports two processor versions:

- **Processor v1** — The legacy processor, based on the [Azure Event Hubs Go SDK v3](https://github.com/Azure/azure-event-hubs-go) (Event Processor Host). Deprecated since 9.3.0 and planned for removal in 9.4.0. Only supports `connection_string` authentication.
- **Processor v2** — The default processor since 9.3.0, based on the modern [Azure SDK for Go](https://github.com/Azure/azure-sdk-for-go). Designed as a drop-in replacement for v1. Supports `connection_string`, `client_secret`, and `managed_identity` authentication.

Use the [`processor_version`](#_processor_version) setting to select which processor to use.

## Migrating from processor v1 to v2 [_migrating_from_processor_v1_to_v2]

Processor v2 is a drop-in replacement for v1. In most cases, existing v1 configurations work with v2 without any changes. This section describes what happens during the migration and the adjustments you may need to make.

:::{important}
Processor v1 is deprecated since 9.3.0 and is planned for removal in 9.4.0. Users should migrate to v2 before upgrading to 9.4.0.
:::

### Checkpoint migration [_checkpoint_migration]

When switching from v1 to v2, the input automatically migrates checkpoint data so that event processing resumes from where v1 left off. No events are reprocessed.

This behavior is controlled by the [`migrate_checkpoint`](#_migrate_checkpoint) setting, which defaults to `true`. On startup, the input reads the v1 checkpoint blobs from the storage container and writes them in the v2 format. If v2 checkpoints already exist for a partition, the migration is skipped for that partition.

You can verify the migration by checking the Filebeat logs for messages like:

```
migrating checkpoint v1 information to v2
migrated checkpoint v1 information to v2
```

### Configuration adjustments [_configuration_adjustments]

**No changes required for basic setups.** If your v1 configuration uses `connection_string` authentication with `storage_account_key`, v2 auto-constructs a `storage_account_connection_string` from the storage account name and key. Your existing configuration continues to work as-is.

However, you should plan to make the following adjustments:

| v1 setting | v2 replacement | Notes |
| --- | --- | --- |
| `storage_account_key` | `storage_account_connection_string` | v2 prefers a full connection string. The key still works (auto-constructed) but is deprecated for v2. |
| `resource_manager_endpoint` | `authority_host` | For sovereign clouds (China, Government, etc.). |

### New capabilities in v2 [_new_capabilities_in_v2]

Processor v2 introduces several new features not available in v1:

- **Additional authentication methods**: `client_secret` and `managed_identity` auth types.
- **WebSocket transport**: Set `transport: "websocket"` to connect through HTTP proxies or when port 5671 (AMQP) is blocked.
- **Tunable batching**: Configure `partition_receive_timeout` and `partition_receive_count` to control how events are batched.
- **Partition claiming interval**: Configure `processor_update_interval` to control how often the processor attempts to claim partitions.

### Pinning to v1 temporarily [_pinning_to_v1_temporarily]

To continue using v1 while planning the migration, explicitly set `processor_version: "v1"` in your configuration:

```yaml
filebeat.inputs:
- type: azure-eventhub
  processor_version: "v1"
  # ... rest of your configuration
```

:::{warning}
Processor v1 is planned for removal in 9.4.0. This is only a temporary measure to give you time to validate v2 in your environment.
:::

## Example configurations

### Connection string authentication (processor v1)

**Note:** Processor v1 only supports connection string authentication.

Example configuration using connection string authentication with processor v1:

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "$Default"
  connection_string: "Endpoint=sb://your-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=your-shared-access-key"
  storage_account: "your-storage-account"
  storage_account_key: "your-storage-account-key"
  storage_account_container: "your-storage-container"
  processor_version: "v1"
```

### Connection string authentication (processor v2)

Example configuration using connection string authentication with processor v2:

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "$Default"
  auth_type: "connection_string"
  connection_string: "Endpoint=sb://your-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=your-shared-access-key"
  storage_account: "your-storage-account"
  storage_account_connection_string: "DefaultEndpointsProtocol=https;AccountName=your-storage-account;AccountKey=your-storage-account-key;EndpointSuffix=core.windows.net"
  storage_account_container: "your-storage-container"
```

### Client secret authentication (processor v2)

```{applies_to}
stack: ga 9.3.0+
```

Example configuration using Azure Active Directory service principal authentication with processor v2:

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "$Default"
  auth_type: "client_secret"
  eventhub_namespace: "your-namespace.servicebus.windows.net"
  tenant_id: "your-tenant-id"
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  storage_account: "your-storage-account"
  storage_account_container: "your-storage-container"
```

:::{note}
When using `client_secret` authentication, the service principal must have the appropriate Azure RBAC permissions. See [Required permissions](#_required_permissions) for details.
:::

### Managed identity authentication (processor v2)

```{applies_to}
stack: ga 9.2.6+
```

Example configuration using Azure Managed Identity authentication with processor v2. This is ideal for workloads running on Azure VMs, Azure Container Apps, Azure Kubernetes Service (AKS), or other Azure services that support managed identities.

:::{important}
Available starting from {{filebeat}} 9.2.6 and later, 9.3.1 and later, 9.4.0 and later, and {{stack}} 8.19.12 and later.
:::

**System-assigned managed identity:**

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "$Default"
  auth_type: "managed_identity"
  eventhub_namespace: "your-namespace.servicebus.windows.net"
  storage_account: "your-storage-account"
  storage_account_container: "your-storage-container"
```

**User-assigned managed identity:**

```yaml
filebeat.inputs:
- type: azure-eventhub
  eventhub: "insights-operational-logs"
  consumer_group: "$Default"
  auth_type: "managed_identity"
  eventhub_namespace: "your-namespace.servicebus.windows.net"
  managed_identity_client_id: "your-user-assigned-identity-client-id"
  storage_account: "your-storage-account"
  storage_account_container: "your-storage-container"
```

:::{note}
When using `managed_identity` authentication, the managed identity must have the appropriate Azure RBAC permissions. Refer to [Required permissions](#_required_permissions) for details.
:::

## Authentication [_authentication]

The azure-eventhub input supports multiple authentication methods. The [`auth_type` configuration option](#_auth_type) controls the authentication method used for both Event Hub and Storage Account.

### Authentication types

The following authentication types are supported:

- **`connection_string`** (default if `auth_type` is not specified): Uses Azure Event Hubs and Storage Account connection strings.
- {applies_to}`stack: ga 9.3.0` **`client_secret`**: Uses Azure Active Directory service principal with client secret credentials.
- {applies_to}`stack: ga 9.2.6+` {applies_to}`stack: ga 8.19.12+` **`managed_identity`**: Uses Azure Managed Identity. Supports both system-assigned and user-assigned managed identities. Available starting from {{filebeat}} 9.2.6 and later, 9.3.1 and later, 9.4.0 and later, and {{stack}} 8.19.12 and later.

### Required permissions [_required_permissions]

When using `client_secret` or `managed_identity` authentication, the identity (service principal or managed identity) needs the following Azure RBAC permissions:

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

## Configuration options [_configuration_options]

The `azure-eventhub` input supports the following configuration options:

### `eventhub` [_eventhub]

The name of the eventhub users would like to read from, field required.

### `consumer_group` [_consumer_group]

Optional, we recommend using a dedicated consumer group for the azure input. Reusing consumer groups among non-related consumers can cause unexpected behavior and possibly lost events.

### `auth_type` [_auth_type]

```{applies_to}
stack: ga 9.3.0
```

Specifies the authentication method to use for both Event Hub and Storage Account. If not specified, defaults to `connection_string` for backwards compatibility.

Valid values include:
- `connection_string` (default): Uses connection string authentication. You _must_ provide a [`connection_string`](#_connection_string).
- `client_secret`: Uses Azure Active Directory service principal with client secret credentials.
- {applies_to}`stack: ga 9.2.6+` {applies_to}`stack: ga 8.19.12+` `managed_identity`: Uses Azure Managed Identity. Ideal for workloads running on Azure infrastructure. Available starting from {{filebeat}} 9.2.6 and later, 9.3.1 and later, 9.4.0 and later, and {{stack}} 8.19.12 and later.

### `connection_string` [_connection_string]

The connection string required to communicate with Event Hubs when using `connection_string` authentication. For more information, refer to [Get an Azure Event Hubs connection string](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-get-connection-string).

This option is required if:

* `auth_type` is set to `connection_string`
* `auth_type` is not specified (in which case it defaults to `connection_string` for backwards compatibility)

A Blob Storage account is required to store, retrieve, or update the offset or state of the Event Hub messages. This means that after stopping Filebeat it can resume from where it stopped processing messages.

### `eventhub_namespace` [_eventhub_namespace]

```{applies_to}
stack: ga 9.3.0
```

The fully qualified namespace for the Event Hub. Required when using credential-based authentication methods (such as `client_secret` or `managed_identity`). Not required when using `connection_string` authentication, as the namespace is embedded in the connection string. Format: `your-eventhub-namespace.servicebus.windows.net`

### `tenant_id` [_tenant_id]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory tenant ID. Required when using `client_secret` authentication for Event Hub or Storage Account.

### `client_id` [_client_id]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory application (client) ID. Required when using `client_secret` authentication for Event Hub or Storage Account.

### `client_secret` [_client_secret]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory application client secret. Required when using `client_secret` authentication for Event Hub or Storage Account.

### `authority_host` [_authority_host]

```{applies_to}
stack: ga 9.3.0
```

The Azure Active Directory authority host. Optional when using `client_secret` or `managed_identity` authentication. Defaults to Azure Public Cloud (`https://login.microsoftonline.com`).

Supported values:
- `https://login.microsoftonline.com` (Azure Public Cloud - default)
- `https://login.microsoftonline.us` (Azure Government)
- `https://login.chinacloudapi.cn` (Azure China)

### `managed_identity_client_id` [_managed_identity_client_id]

```{applies_to}
stack: ga 9.2.6+
```

The client ID of a user-assigned managed identity. Optional when using `managed_identity` authentication. If not specified, the system-assigned managed identity is used.

Use this option when:
- Your Azure resource has multiple user-assigned managed identities and you need to specify which one to use.
- You want to use a user-assigned managed identity instead of the system-assigned managed identity.

:::{important}
Available starting from {{filebeat}} 9.2.6 and later, 9.3.1 and later, 9.4.0 and later, and {{stack}} 8.19.12 and later.
:::

### `storage_account` [_storage_account]

The name of the storage account. Required.

### `storage_account_key` [_storage_account_key]

The storage account key. Used to authorize access to data in your storage account.

:::{note}
This option is used by processor v1 only. Processor v2 prefers [`storage_account_connection_string`](#_storage_account_connection_string). When using processor v2 with `storage_account_key`, the input auto-constructs a connection string from the storage account name and key for backward compatibility — but you should migrate to `storage_account_connection_string`.
:::

### `storage_account_connection_string` [_storage_account_connection_string]

```{applies_to}
stack: ga 8.15.1
```

The connection string for the storage account. Required when using processor v2 with `connection_string` authentication.

Format: `DefaultEndpointsProtocol=https;AccountName=<name>;AccountKey=<key>;EndpointSuffix=core.windows.net`

When using `client_secret` or `managed_identity` authentication, this option is not required — the storage account uses the same credentials as the Event Hub.

### `storage_account_container` [_storage_account_container]

Optional, the name of the storage account container you would like to store the offset information in.

### `resource_manager_endpoint` [_resource_manager_endpoint]

Optional, by default we are using the azure public environment, to override, users can provide a specific resource manager endpoint in order to use a different azure environment. Ex: [https://management.chinacloudapi.cn/](https://management.chinacloudapi.cn/) for azure ChinaCloud [https://management.microsoftazure.de/](https://management.microsoftazure.de/) for azure GermanCloud [https://management.azure.com/](https://management.azure.com/) for azure PublicCloud [https://management.usgovcloudapi.net/](https://management.usgovcloudapi.net/) for azure USGovernmentCloud Users can also use this in case of a Hybrid Cloud model, where one may define their own endpoints.

:::{note}
This option is used by processor v1 only. Processor v2 uses [`authority_host`](#_authority_host) instead for sovereign cloud environments.
:::

### `processor_version` [_processor_version]

```{applies_to}
stack: ga 8.15.1
```

Controls which processor version to use. Valid values are `v1` and `v2`. Default was `v1` in 8.15.1–9.2.x; changed to `v2` in 9.3.0.

Processor v1 is deprecated and planned for removal in 9.4.0. Refer to [Migrating from processor v1 to v2](#_migrating_from_processor_v1_to_v2) for details.

### `migrate_checkpoint` [_migrate_checkpoint]

```{applies_to}
stack: ga 8.15.1
```

Controls whether the input should migrate checkpoint information from v1 to v2 format on startup. Default is `true`. Processor v2 only.

When enabled, the input reads existing v1 checkpoint blobs and writes them in the v2 format so that event processing resumes from where v1 left off. If v2 checkpoints already exist for a partition, the migration is skipped for that partition.

### `processor_update_interval` [_processor_update_interval]

```{applies_to}
stack: ga 8.15.1
```

Controls how often the processor attempts to claim partitions. Default is `10s`. Processor v2 only.

### `processor_start_position` [_processor_start_position]

```{applies_to}
stack: ga 8.15.1
```

Controls the start position for all partitions when no checkpoint exists. Valid values are `earliest` and `latest`. Default is `earliest`. Processor v2 only.

### `partition_receive_timeout` [_partition_receive_timeout]

```{applies_to}
stack: ga 8.15.1
```

Controls the maximum time to wait for events per batch. Works together with [`partition_receive_count`](#_partition_receive_count) — the partition client waits up to `partition_receive_timeout` or for at least `partition_receive_count` events, then returns the events it has received. Default is `5s`. Processor v2 only.

### `partition_receive_count` [_partition_receive_count]

```{applies_to}
stack: ga 8.15.1
```

Controls the maximum number of events per batch. Works together with [`partition_receive_timeout`](#_partition_receive_timeout) — the partition client waits up to `partition_receive_timeout` or for at least `partition_receive_count` events, then returns the events it has received. Default is `100`. Processor v2 only.

### `transport` [_transport]

```{applies_to}
stack: ga 9.3.0
```

Controls the transport type for the Event Hub connection. Valid values are `amqp` (default) and `websocket`. Processor v2 only.

Use `websocket` when connecting through HTTP proxies or when port 5671 (AMQP) is blocked.

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
