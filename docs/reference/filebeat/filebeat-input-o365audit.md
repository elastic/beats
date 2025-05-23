---
navigation_title: "Office 365 Management Activity API"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-o365audit.html
---

# Office 365 Management Activity API input [filebeat-input-o365audit]

:::{admonition} Deprecated in 8.14.0
The o365audit input is deprecated. For collecting Microsoft Office 365 log data, please use the [Microsoft 365](integration-docs://reference/o365/index.md) integration package. For more complex or user-specific use cases, similar functionality can be achieved using the [`CEL input`](/reference/filebeat/filebeat-input-cel.md) .
:::

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Use the `o365audit` input to retrieve audit messages from Office 365 and Azure AD activity logs. These are the same logs that are available under *Audit* *log* *search* in the *Security* *and* *Compliance* center.

A single input instance can be used to fetch events for multiple tenants as long as a single application is configured to access all tenants. Certificate-based authentication is recommended in this scenario.

This input doesn’t perform any transformation on the incoming messages, notably no [Elastic Common Schema fields](ecs://reference/index.md) are populated, and some data is encoded as arrays of objects, which are difficult to query in Elasticsearch. You probably want to use the [Office 365 module](/reference/filebeat/filebeat-module-o365.md) instead.

Example configuration:

```yaml
filebeat.inputs:
- type: o365audit
  application_id: my-application-id
  tenant_id: my-tenant-id
  client_secret: my-client-secret
```

Multi-tenancy and certificate-based authentication is also supported:

```yaml
filebeat.inputs:
- type: o365audit
  application_id: my-application-id
  tenant_id:
    - tenant-id-A
    - tenant-id-B
    - tenant-id-C
  certificate: /path/to/cert.pem
  key: /path/to/private.pem
  # key_passphrase: "my key's password"
```

## Configuration options [_configuration_options_14]

The `o365audit` input supports the following configuration options plus the [Common options](#filebeat-input-o365audit-common-options) described later.


#### `application_id` [_application_id]

The Application ID (also known as Client ID) of the Azure application to authenticate as.


#### `tenant_id` [_tenant_id_2]

The tenant ID (also known as Directory ID) whose data is to be fetched. It’s also possible to specify a list of tenants IDs to fetch data from more than one tenant.


#### `content_type` [_content_type_2]

List of content types to fetch. The default is to fetch all known content types:

* Audit.AzureActiveDirectory
* Audit.Exchange
* Audit.SharePoint
* Audit.General
* DLP.All


#### `client_secret` [_client_secret_2]

The client secret used for authentication.


#### `certificate` [_certificate]

Path to the public certificate file used for certificate-based authentication.


#### `key` [_key]

Path to the certificate’s private key file for certificate-based authentication.


#### `key_passphrase` [_key_passphrase]

Passphrase used to decrypt the private key.


#### `api.authentication_endpoint` [_api_authentication_endpoint]

The authentication endpoint used to authorize the Azure app. This is `https://login.microsoftonline.com/` by default, and can be changed to access alternative endpoints.

### `api.resource` [_api_resource]

The API resource to retrieve information from. This is `https://manage.office.com` by default, and can be changed to access alternative endpoints.


### `api.max_retention` [_api_max_retention]

The maximum data retention period to support. `168h` by default. Filebeat will fetch all retained data for a tenant when run for the first time.


### `api.poll_interval` [_api_poll_interval]

The interval to wait before polling the API server for new events. Default `3m`.


### `api.max_requests_per_minute` [_api_max_requests_per_minute]

The maximum number of requests to perform per minute, for each tenant. The default is `2000`, as this is the server-side limit per tenant.


### `api.max_query_size` [_api_max_query_size]

The maximum time window that API allows in a single query. Defaults to `24h` to match Microsoft’s documented limit.


### `api.preserve_original_event` [_api_preserve_original_event]

Controls whether the original o365 audit object will be kept in `event.original` or not. Defaults to `false`.



## Common options [filebeat-input-o365audit-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_19]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_19]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: o365audit
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-o365audit-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: o365audit
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-o365audit]

If this option is set to true, the custom [fields](#filebeat-input-o365audit-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_19]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_19]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_19]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_19]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_19]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


