---
navigation_title: "Streaming"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-streaming.html
---

# Streaming Input [filebeat-input-streaming]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::



The `streaming` input reads messages from a streaming data source, for example a websocket server. This input uses the `CEL engine` and the `mito` library internally to parse and process the messages. Having support for `CEL` allows you to parse and process the messages in a more flexible way. It has many similarities with the `cel` input as to how the `CEL` programs are written but differs in the way the messages are read and processed. Currently websocket server or API endpoints, and the Crowdstrike Falcon streaming API are supported.

The websocket streaming input supports:

* Auth

    * Basic
    * Bearer
    * Custom
    * OAuth2.0


::::{note}
The `streaming` input websocket handler does not currently support XML messages. Auto-reconnects are also not supported at the moment so reconnection will occur on input restart.
::::


The Crowdstrike streaming input requires OAuth2.0 as described in the Crowdstrike documentation for the API. When using the Crowdstrike streaming type, the `crowdstrike_app_id` configuration field must be set. This field specifies the `appId` parameter sent to the Crowdstrike API. See the Crowdstrike documentation for details.

The `stream_type` configuration field specifies which type of streaming input to use, "websocket" or "crowdstrike". If it is not set, the input defaults to websocket streaming  .

## Execution [_execution_3]

The execution environment provided for the input includes includes the functions, macros, and global variables provided by the mito library. A single JSON object is provided as an input accessible through a `state` variable. `state` contains a `response` map field and may contain arbitrary other fields configured via the input’s `state` configuration. If the CEL program saves cursor states between executions of the program, the configured `state.cursor` value will be replaced by the saved cursor prior to execution.

On start the `state` will be something like this:

```json
{
    "response": { ... },
    "cursor": { ... },
    ...
}
```

The `streaming` input websocket handler creates a `response` field in the state map and attaches the websocket message to this field. All `CEL` programs written should act on this `response` field. Additional fields may be present at the root of the object and if the program tolerates it, the cursor value may be absent. Only the cursor is persisted over restarts, but all fields in state are retained between iterations of the processing loop except for the produced events array, see below.

If the cursor is present the program should process or filter out responses based on its value. If cursor is not present all responses should be processed as per the program’s logic.

After completion of a program’s execution it should return a single object with a structure looking like this:

```json
{
    "events": [ <1>
        {...},
        ...
    ],
    "cursor": [ <2>
        {...},
        ...
    ]
}
```

1. The `events` field must be present, but may be empty or null. If it is not empty, it must only have objects as elements. The field could be an array or a single object that will be treated as an array with a single element. This depends completely on the streaming data source. The `events` field is the array of events to be published to the output. Each event must be a JSON object.
2. If `cursor` is present it must be either be a single object or an array with the same length as events; each element *i* of the `cursor` will be the details for obtaining the events at and beyond event *i* in the `events` array. If the `cursor` is a single object, it will be the details for obtaining events after the last event in the `events` array and will only be retained on successful publication of all the events in the `events` array.


Example configurations:

```yaml
filebeat.inputs:
# Read and process simple websocket messages from a local websocket server
- type: streaming
  url: ws://localhost:443/v1/stream
  program: |
    bytes(state.response).decode_json().as(inner_body,{
      "events": {
        "message":  inner_body.encode_json(),
      }
    })
```

```yaml
filebeat.inputs:
# Read and process events from the Crowdstrike Falcon Hose API
- type: streaming
  stream_type: crowdstrike
  url: https://api.crowdstrike.com/sensors/entities/datafeed/v2
  auth:
    client_id: a23fcea2643868ef1a41565a1a8a1c7c
    client_secret: c3VwZXJzZWNyZXRfY2xpZW50X3NlY3JldF9zaGhoaGgK
    token_url: https://api.crowdstrike.com/oauth2/token
  crowdstrike_app_id: my_app_id
  program: |
    state.response.decode_json().as(body,{
      "events": [body],
      ?"cursor": has(body.?metadata.offset) ?
        optional.of({"offset": body.metadata.offset})
      :
        optional.none(),
    })
```


## Debug state logging [_debug_state_logging_2]

The Websocket input will log the complete state when logging at the DEBUG level before and after CEL evaluation. This will include any sensitive or secret information kept in the `state` object, and so DEBUG level logging should not be used in production when sensitive information is retained in the `state` object. See [`redact`](#streaming-state-redact) configuration parameters for settings to exclude sensitive fields from DEBUG logs.


## Authentication [_authentication]

The websocket streaming input supports authentication via Basic token authentication, Bearer token authentication, authentication via a custom auth config and OAuth2 based authentication. Unlike REST inputs Basic Authentication contains a basic auth token, Bearer Authentication contains a bearer token and custom auth contains any combination of custom header and value. These token/key values are are added to the request headers and are not exposed to the `state` object. The custom auth configuration is useful for constructing requests that require custom headers and values for authentication. The basic and bearer token configurations will always use the `Authorization` header and prepend the token with `Basic` or `Bearer` respectively.

Example configurations with authentication:

```yaml
filebeat.inputs:
- type: streaming
  auth.basic_token: "dXNlcjpwYXNzd29yZA=="
  url: wss://localhost:443/_stream
```

```yaml
filebeat.inputs:
- type: streaming
  auth.bearer_token: "dXNlcjpwYXNzd29yZA=="
  url: wss://localhost:443/_stream
```

```yaml
filebeat.inputs:
- type: streaming
  auth.custom:
    header: "x-api-key"
    value: "dXNlcjpwYXNzd29yZA=="
  url: wss://localhost:443/_stream
```

```yaml
filebeat.inputs:
- type: streaming
  auth.custom:
    header: "Auth"
    value: "Bearer dXNlcjpwYXNzd29yZA=="
  url: wss://localhost:443/_stream
```

The crowdstrike streaming input requires OAuth2.0 authentication using a client ID, client secret and a token URL. These values are not exposed to the `state` object. OAuth2.0 scopes and endpoint parameters are available via the `auth.scopes` and `auth.endpoint_params` config parameters.

```yaml
filebeat.inputs:
- type: streaming
  stream_type: crowdstrike
  auth:
    client_id: a23fcea2643868ef1a41565a1a8a1c7c
    client_secret: c3VwZXJzZWNyZXRfY2xpZW50X3NlY3JldF9zaGhoaGgK
    token_url: https://api.crowdstrike.com/oauth2/token
```


## Websocket OAuth2.0 [_websocket_oauth2_0]

The `websocket` streaming input supports OAuth2.0 authentication. The `auth` configuration field is used to specify the OAuth2.0 configuration. These values are not exposed to the `state` object.

The `auth` configuration field has the following subfields:

* `client_id`: The client ID to use for OAuth2.0 authentication.
* `client_secret`: The client secret to use for OAuth2.0 authentication.
* `token_url`: The token URL to use for OAuth2.0 authentication.
* `scopes`: The scopes to use for OAuth2.0 authentication.
* `endpoint_params`: The endpoint parameters to use for OAuth2.0 authentication.
* `auth_style`: The authentication style to use for OAuth2.0 authentication. If left unset, the style will be automatically detected.
* `token_expiry_buffer`: Minimum valid time remaining before attempting an OAuth2 token renewal. The default value is `2m`.

**Explanations for `auth_style` and `token_expiry_buffer`:**

* `auth_style`: The authentication style to use for OAuth2.0 authentication which determines how the values of sensitive information like `client_id` and `client_secret` are sent in the token request. The default style value is automatically inferred and used appropriately if no value is provided. The `auth_style` configuration field is optional and can be used to specify the authentication style to use for OAuth2.0 authentication. The `auth_style` configuration field supports the following configurable values:

    * `in_header`: The `client_id` and `client_secret` is sent in the header as a base64 encoded `Authorization` header.
    * `in_params`: The `client_id` and `client_secret` is sent in the request body along with the other OAuth2 parameters.

* `token_expiry_buffer`: The token expiry buffer to use for OAuth2.0 authentication. The `token_expiry_buffer` is used as a safety net to ensure that the token does not expire before the input can refresh it. The `token_expiry_buffer` configuration field is optional. If the `token_expiry_buffer` configuration field is not set, the default value of `2m` is used.

::::{note}
We recommend leaving the `auth_style` configuration field unset (automatically inferred internally) for most scenarios, except where manual intervention is required.
::::


```yaml
filebeat.inputs:
- type: streaming
  auth:
    client_id: a23fcea2643868ef1a41565a1a8a1c7c
    client_secret: c3VwZXJzZWNyZXRfY2xpZW50X3NlY3JldF9zaGhoaGgK
    token_url: https://api.sample-url.com/oauth2/token
    scopes: ["read", "write"]
    endpoint_params:
      param1: value1
      param2: value2
    auth_style: in_params
    token_expiry_buffer: 5m
  url: wss://localhost:443/_stream
```

## Keep Alive configuration

The `streaming` input currently supports keep-alive configuration options for streams of `type: websocket`. Use these configuration options to further optimize the stability
of your WebSocket connections and prevent them from idling out.

The `keep_alive` setting has the following configuration options:

* `enable`: Indicates whether Keep-Alive is enabled. By default, this is set to `false`.
* `interval`: Interval between Keep-Alive messages, expressed as a time duration value. The default value is `30s`.
* `write_control_deadline`: Deadline for writing control frames, like `PING`, `PONG`, or `CLOSE`, on a WebSocket connection. The timeout, expressed as a time duration value, helps prevent indefinite blocking when the server or client is not responding to control frame requests. The default value is `10s`.
   
::::{note}
Don't use the `blanket_retries` and `infinite_retries` configuration options together with the `keep_alive` settings. The purpose of `keep_alive` is to keep the connection open so you don't need to `retry` and reconnect all the time. In some scenarios `keep_alive` might not work if the host WebSocket server is not configured to handle `ping` frames.
::::

## Input state [input-state-streaming]

The `streaming` input keeps a runtime state between every message received. This state can be accessed by the CEL program and may contain arbitrary objects. The state must contain a `response` map and may contain any object the user wishes to store in it. All objects are stored at runtime, except `cursor`, which has values that are persisted between restarts.


## Configuration options [_configuration_options_17]

The `streaming` input supports the following configuration options plus the [Common options](#filebeat-input-streaming-common-options) described later.


### `stream_type` [stream_type-streaming]

The flavor of streaming to use. This may be either "websocket", "crowdstrike", or unset. If the field is unset, websocket streaming is used.


### `program` [program-streaming]

The CEL program that is executed on each message received. This field should ideally be present but if not the default program given below is used.

```yaml
program: |
  bytes(state.response).decode_json().as(inner_body,{
    "events": {
      "message":  inner_body.encode_json(),
    }
  })
```


### `url_program` [input-url-program-streaming]

If present, this CEL program is executed before the streaming connection is established using the `state` object, including any stored cursor value. It must evaluate to a valid URL. The returned URL is used to make the streaming connection for processing. The program may use cursor values or other state defined values to customize the URL at runtime.

```yaml
url: ws://testapi:443/v1/streamresults
state:
  initial_start_time: "2022-01-01T00:00:00Z"
url_program: |
  state.url + "?since=" + state.?cursor.since.orValue(state.initial_start_time)
program: |
  bytes(state.response).decode_json().as(inner_body,{
    "events": {
      "message":  inner_body.encode_json(),
    },
    "cursor": {
      "since": inner_body.timestamp
    }
  })
```


### `state` [state-streaming]

`state` is an optional object that is passed to the CEL program on the first execution. It is available to the executing program as the `state` variable. Except for the `state.cursor` field, `state` does not persist over restarts.


### `state.cursor` [cursor-streaming]

The cursor is an object available as `state.cursor` where arbitrary values may be stored. Cursor state is kept between input restarts and updated after each event of a request has been published. When a cursor is used the CEL program must either create a cursor state for each event that is returned by the program, or a single cursor that reflects the cursor for completion of the full set of events.

```yaml
filebeat.inputs:
# Read and process simple websocket messages from a local websocket server
- type: streaming
  url: ws://localhost:443/v1/stream
  program: |
    bytes(state.response).as(body, {
      "events": [body.decode_json().with({
        "last_requested_at": has(state.cursor) && has(state.cursor.last_requested_at) ?
          state.cursor.last_requested_at
        :
          now
      })],
      "cursor": {"last_requested_at": now}
    })
```


### `regexp` [regexp-streaming]

A set of named regular expressions that may be used during a CEL program’s execution using the `regexp` extension library. The syntax used for the regular expressions is [RE2](https://github.com/google/re2/wiki/Syntax).

```yaml
filebeat.inputs:
- type: streaming
  # Define two regular expressions, 'products' and 'solutions' for use during CEL program execution.
  regexp:
    products: '(?i)(Elasticsearch|Beats|Logstash|Kibana)'
    solutions: '(?i)(Search|Observability|Security)'
```


### `redact` [streaming-state-redact]

During debug level logging, the `state` object and the resulting evaluation result are included in logs. This may result in leaking of secrets. In order to prevent this, fields may be redacted or deleted from the logged `state`. The `redact` configuration allows users to configure this field redaction behaviour. For safety reasons if the `redact` configuration is missing a warning is logged.

In the case of no-required redaction an empty `redact.fields` configuration should be used to silence the logged warning.

```yaml
- type: streaming
  redact:
    fields: ~
```

As an example, if a user-constructed Basic Authentication request is used in a CEL program the password can be redacted like so

```yaml
filebeat.inputs:
- type: streaming
  url: ws://localhost:443/_stream
  state:
    user: user@domain.tld
    password: P@$$W0₹D
  redact:
    fields:
      - password
    delete: true
```

Note that fields under the `auth` configuration hierarchy are not exposed to the `state` and so do not need to be redacted. For this reason it is preferable to use these for authentication over the request construction shown above where possible.


### `redact.fields` [_redact_fields_2]

This specifies fields in the `state` to be redacted prior to debug logging. Fields listed in this array will be either replaced with a `*` or deleted entirely from messages sent to debug logs.


### `redact.delete` [_redact_delete_2]

This specifies whether fields should be replaced with a `*` or deleted entirely from messages sent to debug logs. If delete is `true`, fields will be deleted rather than replaced.


### `retry` [retry-streaming]

The `retry` configuration allows the user to specify the number of times the input should attempt to reconnect to the streaming data source in the event of a connection failure. The default value is `nil` which means no retries will be attempted. It has a `wait_min` and `wait_max` configuration which specifies the minimum and maximum time to wait between retries. It also supports blanket retries and infinite retries via the `blanket_retires` and `infinite_retries` configuration options. These are set to `false` by default.

```yaml
filebeat.inputs:
- type: streaming
  url: ws://localhost:443/_stream
  program: |
    bytes(state.response).decode_json().as(inner_body,{
      "events": {
        "message":  inner_body.encode_json(),
      }
    })
  retry:
    max_attempts: 5
    wait_min: 1s
    wait_max: 10s
    blanket_retries: false
    infinite_retries: false
```


### `retry.max_attempts` [_retry_max_attempts]

The maximum number of times the input should attempt to reconnect to the streaming data source in the event of a connection failure. The default value is `5` which means a maximum of 5 retries will be attempted.


### `retry.wait_min` [_retry_wait_min]

The minimum time to wait between retries. This ensures that retries are spaced out enough to give the system time to recover or resolve transient issues, rather than bombarding the system with rapid retries. For example, `wait_min` might be set to 1 second, meaning that even if the calculated backoff is less than this, the client will wait at least 1 second before retrying. The default value is `1` second.


### `retry.wait_max` [_retry_wait_max]

The maximum time to wait between retries. This prevents the retry mechanism from becoming too slow, ensuring that the client does not wait indefinitely between retries. This is crucial in systems where timeouts or user experience are critical. For example, `wait_max` might be set to 10 seconds, meaning that even if the calculated backoff is greater than this, the client will wait at most 10 seconds before retrying. The default value is `30` seconds.


### `retry.blanket_retries` [_retry_blanket_retries]

Normally the input will only retry when a connection error is found to be retryable based on the error type and the RFC 6455 error codes defined by the websocket protocol. If `blanket_retries` is set to `true` (`false` by default) the input will retry on any error. This is not recommended unless the user is certain that all errors are transient and can be resolved by retrying.


### `retry.infinite_retries` [_retry_infinite_retries]

Normally the input will only retry a maximum of `max_attempts` times. If `infinite_retries` is set to `true` (`false` by default) the input will retry indefinitely. This is not recommended unless the user is certain that the connection will eventually succeed.


## `timeout` [_timeout]

Timeout is the maximum amount of time the websocket dialer will wait for a connection to be established. The default value is `180` seconds.


### `proxy_url` [_proxy_url]

This specifies the forward proxy URL to use for the connection. The `proxy_url` configuration is optional and can be used to configure the proxy settings for the connection. The `proxy_url` default value is set by `http.ProxyFromEnvironment` which reads the `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.


### `proxy_headers` [_proxy_headers]

This specifies the headers to be sent to the proxy server. The `proxy_headers` configuration is optional and can be used to configure the headers to be sent to the proxy server.


### `ssl` [_ssl_3]

This specifies the SSL configuration for the connection. The `ssl` configuration is optional and can be used to configure the SSL settings for the connection. The `ssl` configuration has the following subfields:

* `certificate_authorities`: A list of root certificates to use for verifying the server’s certificate.
* `certificate`: The (PEM encoded) certificate to use for client authentication.
* `key`: The (PEM encoded) private key to use for client authentication.

If this is a self-signed certificate, the `certificate_authorities` field should be set to the certificate itself.


## Metrics [_metrics_14]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `url` | URL of the input resource. |
| `cel_eval_errors` | Number of errors encountered during cel program evaluation. |
| `errors_total` | Number of errors encountered over the life cycle of the input. |
| `batches_received_total` | Number of event arrays received. |
| `batches_published_total` | Number of event arrays published. |
| `received_bytes_total` | Number of bytes received over the life cycle of the input. |
| `events_received_total` | Number of events received. |
| `events_published_total` | Number of events published. |
| `write_control_errors` | Number of errors encountered for write control operations. |
| `cel_processing_time` | Histogram of the elapsed successful CEL program processing times in nanoseconds. |
| `batch_processing_time` | Histogram of the elapsed successful batch processing times in nanoseconds (time of receipt to time of ACK for non-empty batches). |
| `ping_message_send_time` | Histogram of the elapsed successful ping message send times in nanoseconds. |
| `pong_message_received_time` | Histogram of the elapsed successful pong message receive times in nanoseconds. |


## Developer tools [_developer_tools_2]

A stand-alone CEL environment that implements the majority of the streaming input’s Comment Expression Language functionality is available in the [Elastic Mito](https://github.com/elastic/mito) repository. This tool may be used to help develop CEL programs to be used by the input. Installation is available from source by running `go install github.com/elastic/mito/cmd/mito@latest` and requires a Go toolchain.


## Common options [filebeat-input-streaming-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_24]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_23]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: streaming
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-streaming-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: streaming
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-streaming]

If this option is set to true, the custom [fields](#filebeat-input-streaming-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_23]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_23]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_23]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_23]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_23]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.

::::{note}
The `streaming` input is currently tagged as experimental and might have bugs and other issues. Please report any issues on the [Github](https://github.com/elastic/beats) repository.
::::
