---
navigation_title: "Salesforce"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-salesforce.html
---

# Salesforce input [filebeat-input-salesforce]


Use the `salesforce` input to monitor Salesforce events either via the [Salesforce EventLogFile (ELF) API](https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_eventlogfile.htm) or the [Salesforce Real-time event monitoring API](https://developer.salesforce.com/blogs/2020/05/introduction-to-real-time-event-monitoring). Both use REST API (to execute SOQL queries in the Salesforce instance) under the hood to query the relevant objects to fetch the events.

The Salesforce input maintains cursor states between requests to track the last event retrieved in each execution. These cursor states are passed to the next event monitoring execution to resume fetching events from the last known position. The cursor states allow the input to pick up where it left off and provide control over the behavior of the input.

Here are some supported authentication methods and event monitoring methods:

* Authentication methods

    * OAuth2

        * User-Password flow
        * JWT Bearer flow

* Event monitoring methods

    * EventLogFile (ELF) using REST API
    * REST API for objects (For monitoring real-time events)


Here are some key points about how cursors are used in the Salesforce input:

* Separate cursor states are maintained for each configured event monitoring method (`event_log_file` and `object`).
* The cursor state stores the unique identifier of the last event retrieved, based on the `cursor.field` specified in the configuration.
* On the first run, the `query.default` is used to fetch an initial set of events.
* On subsequent runs, the `query.value` template is populated with the cursor state to fetch events since the last execution.
* If the input is restarted, it will resume from the last persisted cursor state rather than starting over from scratch.

Using cursors allows the Salesforce input to reliably keep track of its progress and avoid missing or duplicating events across executions. The cursor field should be chosen carefully to have a monotonically increasing value for each new event.

Event Monitoring methods are highly configurable and can be used to monitor any supported object or event log file. The input can be configured to monitor multiple objects or event log files at the same time.

Example configuration:

```yaml
filebeat.inputs:
  - type: salesforce
    enabled: true
    version: 56
    auth.oauth2:
      user_password_flow:
        enabled: true
        client.id: client-id
        client.secret: client-secret
        token_url: https://instance-id.develop.my.salesforce.com
        username: salesforce-instance@user.in
        password: salesforce-instance-password
      jwt_bearer_flow:
        enabled: true
        client.id: client-id
        client.username: salesforce-instance@user.in
        client.key_path: server_client.key
        url: https://login.salesforce.com
    url: https://instance-id.develop.my.salesforce.com
    event_monitoring_method:
      event_log_file:
        enabled: true
        interval: 1h
        query:
          default: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' ORDER BY CreatedDate ASC NULLS FIRST"
          value: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.last_event_time ]] ORDER BY CreatedDate ASC NULLS FIRST"
        cursor:
          field: "CreatedDate"
      object:
        enabled: true
        interval: 5m
        query:
          default: "SELECT FIELDS(STANDARD) FROM LoginEvent"
          value: "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.object.first_event_time ]]"
        cursor:
          field: "EventDate"
```

## Set up the OAuth App in the Salesforce [_set_up_the_oauth_app_in_the_salesforce]

In order to use this integration, users need to create a new Salesforce Application using OAuth. Follow the steps below to create a connected application in Salesforce:

1. Login to [Salesforce](https://login.salesforce.com/) with the same user credentials that the user wants to collect data with.
2. Click on Setup on the top right menu bar. On the Setup page, search for `App Manager` in the `Search Setup` search box at the top of the page, then select `App Manager`.
3. Click *New Connected App*.
4. Provide a name for the connected application. This will be displayed in the App Manager and on its App Launcher tile.
5. Enter the API name. The default is a version of the name without spaces. Only letters, numbers, and underscores are allowed. If the original app name contains any other characters, edit the default name.
6. Enter the contact email for Salesforce.
7. Under the API (Enable OAuth Settings) section of the page, select *Enable OAuth Settings*.
8. In the Callback URL, enter the Instance URL (Please refer to `Salesforce Instance URL`).
9. Select the following OAuth scopes to apply to the connected app:

    * Manage user data via APIs (api).
    * Perform requests at any time (refresh_token, offline_access).
    * (Optional) In case of data collection, if any permission issues arise, add the Full access (full) scope.

10. Select *Require Secret for the Web Server Flow* to require the app’s client secret in exchange for an access token.
11. Select *Require Secret for Refresh Token Flow* to require the app’s client secret in the authorization request of a refresh token and hybrid refresh token flow.
12. Click Save. It may take approximately 10 minutes for the changes to take effect.
13. Click Continue and then under API details, click Manage Consumer Details. Verify the user account using the Verification Code.
14. Copy `Consumer Key` and `Consumer Secret` from the Consumer Details section, which should be populated as values for Client ID and Client Secret respectively in the configuration.

For more details on how to create a Connected App, refer to the Salesforce documentation [here](https://help.salesforce.com/apex/HTViewHelpDoc?id=connected_app_create.htm).

::::{note}
**Enabling real-time events**

To get started with [real-time](https://developer.salesforce.com/blogs/2020/05/introduction-to-real-time-event-monitoring) events, head to setup and into the quick find search for *Event Manager*. Enterprise and Unlimited environments have access to the Logout Event by default, but the remainder of the events need licensing to access [Shield Event Monitoring](https://help.salesforce.com/s/articleView?id=sf.salesforce_shield.htm&type=5).

::::



## Execution [_execution_2]

The `salesforce` input is a long-running program that retrieves events from a Salesforce instance and sends them to the specified output. The program executes in a loop, fetching events from the Salesforce instance at a preconfigured interval. Each event monitoring method can be configured to run separately and at different intervals. To prevent a sudden spike in memory usage, if multiple event monitoring methods are configured, they are scheduled to run one at a time. Even if the intervals overlap, only one method will be executed randomly, and the other will be executed after the first one completes.

There are two methods to fetch the events from the Salesforce instance:

* `event_log_file`: [EventLogFile](https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_eventlogfile.htm) is a standard object in Salesforce and the event monitoring method uses the REST API under the hood to gather the Salesforce org’s operational events from the object. There is a field EventType that helps distinguish between the types of operational events like — Login, Logout, etc. Uses Salesforce’s query language SOQL to query the object.
* `object`: This method is a general way of retrieving events from a Salesforce instance by using the REST API. It can be used for monitoring [objects](https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_list.htm) in real-time. In real-time event monitoring, subscribing to the events is a common practice, but the events are also stored in Salesforce org (if configured), specifically in big object tables that are preconfigured for each event type. With this method, we query the object using Salesforce’s query language ([SOQL](https://developer.salesforce.com/docs/atlas.en-us.soql_sosl.meta/soql_sosl/sforce_api_calls_soql.htm)). The collection happens at the configured scrape `interval`.

::::{note}
**Salesforce Objects and SOQL Query Field Ordering Limitations**

Each Salesforce Object contains a set of fields, but SOQL queries have restrictions on the fields that can be ordered and the specific ordering method. The Object description on the Salesforce Developers page provides information about these limitations. For instance, the Login Object only allows ordering by the EventDate field in descending order.

When collecting data over time using cursors, the following cursor inputs are available:

* `object.first_event_time`: This cursor input stores the cursor value from the first event encountered during data collection using the object method.
* `object.last_event_time`: This cursor input stores the cursor value from the last event encountered during data collection using the object method.
* `event_log_file.first_event_time`: This cursor input stores the cursor value from the first event encountered during data collection using the event log file method.
* `event_log_file.last_event_time`: This cursor input stores the cursor value from the last event encountered during data collection using the event log file method.

By selecting one of the above cursor inputs, users can collect data from both the object and event log file in the desired order. The cursor configuration can be customized based on the user’s specific requirements.

::::



## Configuration options [_configuration_options_16]

The `salesforce` input supports the following configuration options plus the [Common options](#filebeat-input-salesforce-common-options) described later.


## `enabled` [_enabled_21]

Whether the input is enabled or not. Default: `false`.


## `version` [_version_3]

The version of the Salesforce API to use. Minimum supported version is 46.


## `auth` [_auth]

The authentication settings for the Salesforce instance.


## `auth.oauth2` [_auth_oauth2]

The OAuth2 authentication options for the Salesforce instance.

There are two OAuth2 authentication flows supported:

* `user_password_flow`: User-Password flow
* `jwt_bearer_flow`: JWT Bearer flow


## `auth.oauth2.user_password_flow.enabled` [_auth_oauth2_user_password_flow_enabled]

Whether to use the user-password flow for authentication. Default: `false`.

::::{note}
Only one authentication flow can be enabled at a time.
::::



## `auth.oauth2.user_password_flow.client.id` [_auth_oauth2_user_password_flow_client_id]

The client ID for the user-password flow.


## `auth.oauth2.user_password_flow.client.secret` [_auth_oauth2_user_password_flow_client_secret]

The client secret for the user-password flow.


## `auth.oauth2.user_password_flow.token_url` [_auth_oauth2_user_password_flow_token_url]

The token URL for the user-password flow.


## `auth.oauth2.user_password_flow.username` [_auth_oauth2_user_password_flow_username]

The username for the user-password flow.


## `auth.oauth2.user_password_flow.password` [_auth_oauth2_user_password_flow_password]

The password for the user-password flow.


## `auth.oauth2.jwt_bearer_flow.enabled` [_auth_oauth2_jwt_bearer_flow_enabled]

Whether to use the JWT bearer flow for authentication. Default: `false`.

::::{note}
Only one authentication flow can be enabled at a time.
::::



## `auth.oauth2.jwt_bearer_flow.client.id` [_auth_oauth2_jwt_bearer_flow_client_id]

The client ID for the JWT bearer flow.


## `auth.oauth2.jwt_bearer_flow.client.username` [_auth_oauth2_jwt_bearer_flow_client_username]

The username for the JWT bearer flow.


## `auth.oauth2.jwt_bearer_flow.client.key_path` [_auth_oauth2_jwt_bearer_flow_client_key_path]

The path to the private key file for the JWT bearer flow. The file must be PEM encoded PKCS1 or PKCS8 private key and must have the right permissions set to have read access for the user running the program.


## `auth.oauth2.jwt_bearer_flow.url` [_auth_oauth2_jwt_bearer_flow_url]

The URL for the JWT bearer flow.


## `url` [_url_2]

The URL of the Salesforce instance. Required.


## `resource.timeout` [_resource_timeout_2]

Duration before declaring that the HTTP client connection has timed out. Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`. Default: `30s`.


## `resource.retry.max_attempts` [_resource_retry_max_attempts_2]

The maximum number of retries for the HTTP client. Default: `5`.


## `resource.retry.wait_min` [_resource_retry_wait_min_2]

The minimum time to wait before a retry is attempted. Default: `1s`.


## `resource.retry.wait_max` [_resource_retry_wait_max_2]

The maximum time to wait before a retry is attempted. Default: `60s`.


## `event_monitoring_method` [_event_monitoring_method]

The event monitoring method to use. There are two event monitoring methods supported:

* `event_log_file`: EventLogFile (ELF) using REST API
* `object`: Real-time event monitoring using REST API (objects)


## `event_monitoring_method.event_log_file` [_event_monitoring_method_event_log_file]

The event monitoring method to use — event_log_file. Uses the EventLogFile API to fetch the events from the Salesforce instance.


## `event_monitoring_method.event_log_file.enabled` [_event_monitoring_method_event_log_file_enabled]

Whether to use the EventLogFile API for event monitoring. Default: `false`.


## `event_monitoring_method.event_log_file.interval` [_event_monitoring_method_event_log_file_interval]

The interval to collect the events from the Salesforce instance using the EventLogFile API.


## `event_monitoring_method.event_log_file.query.default` [_event_monitoring_method_event_log_file_query_default]

The default query to fetch the events from the Salesforce instance using the EventLogFile API.

In case the cursor state is not available, the default query will be used to fetch the events from the Salesforce instance. The default query must be a valid SOQL query. If the SOQL query in `event_monitoring_method.event_log_file.query.value` is not valid, the default query will be used to fetch the events from the Salesforce instance.


## `event_monitoring_method.event_log_file.query.value` [_event_monitoring_method_event_log_file_query_value]

The SOQL query to fetch the events from the Salesforce instance using the EventLogFile API but it uses the cursor state to fetch the events from the Salesforce instance. The SOQL query must be a valid SOQL query. If the SOQL query is not valid, the default query will be used to fetch the events from the Salesforce instance.

In case of restarts or subsequent executions, the cursor state will be used to fetch the events from the Salesforce instance. The cursor state is the last event time of the last event fetched from the Salesforce instance. The cursor state is taken from `event_monitoring_method.event_log_file.cursor.field` field for the last event fetched from the Salesforce instance.


## `event_monitoring_method.event_log_file.cursor.field` [_event_monitoring_method_event_log_file_cursor_field]

The field to use to fetch the cursor state from the last event fetched from the Salesforce instance. The field must be a valid field in the SOQL query specified in `event_monitoring_method.event_log_file.query.default` and `event_monitoring_method.event_log_file.query.value` i.e., part of the selected fields in the SOQL query.


## `event_monitoring_method.object` [_event_monitoring_method_object]

The event monitoring method to use — object. Uses REST API to fetch the events directly from the objects from the Salesforce instance.


## `event_monitoring_method.object.enabled` [_event_monitoring_method_object_enabled]

Whether to use the REST API for objects for event monitoring. Default: `false`.


## `event_monitoring_method.object.interval` [_event_monitoring_method_object_interval]

The interval to collect the events from the Salesforce instance using the REST API from objects.


## `event_monitoring_method.object.query.default` [_event_monitoring_method_object_query_default]

The default SOQL query to fetch the events from the Salesforce instance using the REST API from objects.

In case the cursor state is not available, the default query will be used to fetch the events from the Salesforce instance. The default query must be a valid SOQL query. If the SOQL query in `event_monitoring_method.object.query.value` is not valid, the default query will be used to fetch the events from the Salesforce instance.


## `event_monitoring_method.object.query.value` [_event_monitoring_method_object_query_value]

The SOQL query to fetch the events from the Salesforce instance using the REST API from objects but it uses the cursor state to fetch the events from the Salesforce instance. The SOQL query must be a valid SOQL query. If the SOQL query is not valid, the default query will be used to fetch the events from the Salesforce instance.

In case of restarts or subsequent executions, the cursor state will be used to fetch the events from the Salesforce instance. The cursor state is the last event time of the last event fetched from the Salesforce instance. The cursor state is taken from `event_monitoring_method.object.cursor.field` field for the last event fetched from the Salesforce instance.


## `event_monitoring_method.object.cursor.field` [_event_monitoring_method_object_cursor_field]

The field to use to fetch the cursor state from the last event fetched from the Salesforce instance. The field must be a valid field in the SOQL query specified in `event_monitoring_method.object.query.default` and `event_monitoring_method.object.query.value` i.e., part of the selected fields in the SOQL query.


## Common options [filebeat-input-salesforce-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_22]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_21]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: salesforce
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-salesforce-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: salesforce
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-salesforce]

If this option is set to true, the custom [fields](#filebeat-input-salesforce-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_21]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_21]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_21]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_21]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_21]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


