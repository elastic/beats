---
navigation_title: "Azure Event Hub"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-azure-eventhub.html
applies_to:
  stack: ga
  serverless: ga
---

# Azure eventhub input [filebeat-input-azure-eventhub]

Use the `azure-eventhub` input to read messages from an Azure Event Hub. The input uses the [Azure Event Hubs SDK for Go](https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/messaging/azeventhubs) to consume events with load-balanced partition processing across multiple instances.

State such as leases on partitions and checkpoints in the event stream are shared between receivers using an Azure Storage container. For this reason, as a prerequisite to using this input, you must create or use an existing storage account.

## Example configurations

### Connection string authentication

Example configuration using connection string authentication:

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

### Client secret authentication

```{applies_to}
stack: ga 9.3.0+
```

Example configuration using Azure Active Directory service principal authentication:

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

### Managed identity authentication

```{applies_to}
stack: ga 9.2.6+
```

Example configuration using Azure Managed Identity authentication. This is ideal for workloads running on Azure VMs, Azure Container Apps, Azure Kubernetes Service (AKS), or other Azure services that support managed identities.

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

```{applies_to}
stack: deprecated 9.0.0
```

:::{note}
Use [`storage_account_connection_string`](#_storage_account_connection_string) instead. When `storage_account_key` is set together with `storage_account`, the input auto-constructs a connection string for backward compatibility, but this behavior will be removed in a future release.
:::

The storage account key. When using `connection_string` authentication, you can provide either `storage_account_connection_string` (recommended) or `storage_account_key` together with `storage_account` to auto-construct the connection string. Not required when using `client_secret` or `managed_identity` authentication.

### `storage_account_connection_string` [_storage_account_connection_string]

The connection string for the storage account used to store partition ownership and checkpoint information. Required when using `connection_string` authentication. Not required when using `client_secret` or `managed_identity` authentication, as the storage account uses the same credentials as the Event Hub.

### `storage_account_container` [_storage_account_container]

Optional, the name of the storage account container you would like to store the offset information in.

### `resource_manager_endpoint` [_resource_manager_endpoint]

```{applies_to}
stack: deprecated 9.0.0
```

:::{note}
Use [`authority_host`](#_authority_host) instead to control the cloud environment. The `resource_manager_endpoint` option will be removed in a future release.
:::

Optional, by default we are using the azure public environment, to override, users can provide a specific resource manager endpoint in order to use a different azure environment. Ex: [https://management.chinacloudapi.cn/](https://management.chinacloudapi.cn/) for azure ChinaCloud [https://management.microsoftazure.de/](https://management.microsoftazure.de/) for azure GermanCloud [https://management.azure.com/](https://management.azure.com/) for azure PublicCloud [https://management.usgovcloudapi.net/](https://management.usgovcloudapi.net/) for azure USGovernmentCloud Users can also use this in case of a Hybrid Cloud model, where one may define their own endpoints.

### `sanitize_options` [_sanitize_options]

```{applies_to}
stack: deprecated 9.0.0
```

:::{note}
Use [`sanitizers`](#_sanitizers) instead. The `sanitize_options` option will be removed in a future release.
:::

Optional. A list of legacy sanitization options to apply to messages that contain invalid JSON. Supported values: `NEW_LINES`, `SINGLE_QUOTES`.

### `sanitizers` [_sanitizers]

Optional. A list of sanitizers to apply to messages that contain invalid JSON. Each sanitizer has a `type` and an optional `spec` for additional configuration.

Supported sanitizer types:
- `new_lines`: Removes new lines inside JSON strings.
- `single_quotes`: Replaces single quotes with double quotes in JSON strings.
- `replace_all`: Replaces all occurrences of a substring matching a regex `pattern` with a fixed literal string `replacement`. Requires a `spec` with `pattern` and `replacement` fields.

Example:

```yaml
sanitizers:
  - type: new_lines
  - type: single_quotes
  - type: replace_all
    spec:
      pattern: '\[\s*([^\[\]{},\s]+(?:\s+[^\[\]{},\s]+)*)\s*\]'
      replacement: "{}"
```

### `transport` [_transport]

The transport protocol used for the Event Hub connection. Default: `amqp`.

Valid values:
- `amqp` (default): Uses the standard AMQP protocol over port 5671.
- `websocket`: Uses AMQP over WebSockets. Use this when connecting through HTTP proxies or when port 5671 is blocked.

### `processor_update_interval` [_processor_update_interval]

Controls how often the input attempts to claim partitions. Default: `10s`. Minimum: `1s`.

### `processor_start_position` [_processor_start_position]

Controls the start position for all partitions when no checkpoint exists. Default: `earliest`.

Valid values:
- `earliest`: Start reading from the earliest available event in each partition.
- `latest`: Start reading from the latest event, ignoring any events that were sent before the input started.

### `partition_receive_timeout` [_partition_receive_timeout]

Controls the maximum time the partition client waits for events before returning a batch. Works together with `partition_receive_count` — the client returns whichever threshold is reached first. Default: `5s`. Minimum: `1s`.

### `partition_receive_count` [_partition_receive_count]

Controls the minimum number of events the partition client tries to receive before returning a batch. Works together with `partition_receive_timeout` — the client returns whichever threshold is reached first. Default: `100`. Minimum: `1`.

### `migrate_checkpoint` [_migrate_checkpoint]

Controls whether the input migrates checkpoint information from the legacy format to the current format on startup. Default: `true`.

Set this to `true` when upgrading from an older version of {{filebeat}} that used the previous Event Hub processor, to avoid reprocessing events from the beginning of the retention period.

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
