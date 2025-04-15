---
navigation_title: "HTTP Endpoint"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-http_endpoint.html
---

# HTTP Endpoint input [filebeat-input-http_endpoint]


The HTTP Endpoint input initializes a listening HTTP server that collects incoming HTTP POST requests containing a JSON body. The body must be either an object or an array of objects, otherwise a Common Expression Language expression that converts the the JSON body to these types can be provided. Any other data types will result in an HTTP 400 (Bad Request) response. For arrays, one document is created for each object in the array.

gzip encoded request bodies are supported if a `Content-Encoding: gzip` header is sent with the request.

This input can for example be used to receive incoming webhooks from a third-party application or service.

Multiple endpoints may be assigned to a single address and port, and the HTTP Endpoint input will resolve requests based on the URL pattern configuration. If multiple endpoints are configured on a single address they must all have the same TLS configuration, either all disabled or all enabled with identical configurations.

These are the possible response codes from the server.

| HTTP Response Code | Name | Reason |
| --- | --- | --- |
| 200 | OK | Returned on success. |
| 400 | Bad Request | Returned if JSON body decoding fails or if `wait_for_completion_timeout` query validation fails. |
| 401 | Unauthorized | Returned when basic auth, secret header, or HMAC validation fails. |
| 405 | Method Not Allowed | Returned if methods other than POST are used. |
| 406 | Not Acceptable | Returned if the POST request does not contain a body. |
| 415 | Unsupported Media Type | Returned if the Content-Type is not application/json. Or if Content-Encoding is present and is not gzip. |
| 500 | Internal Server Error | Returned if an I/O error occurs reading the request. |
| 503 | Service Unavailable | Returned if the length of the request body would take the total number of in-flight bytes above the configured `max_in_flight_bytes` value. |
| 504 | Gateway Timeout | Returned if a request publication cannot be ACKed within the required timeout. |

The endpoint will enforce end-to-end ACK when a URL query parameter `wait_for_completion_timeout` with a duration is provided. For example `http://localhost:8080/?wait_for_completion_timeout=1m` will wait up to 1 minute for the event to be published to the cluster and then return the user-defined response message. In the case that the publication does not complete within the timeout duration, the HTTP response will have a 504 Gateway Timeout status code. The syntax for durations is a number followed by units which may be h, m and s. No other HTTP query is accepted. If another query parameter is provided or duration syntax is incorrect, the request will fail with an HTTP 400 "Bad Request" status.

Example configurations:

Basic example:

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
```

Custom response example:

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  response_code: 200
  response_body: '{"message": "success"}'
  url: "/"
  prefix: "json"
```

Map request to root of document example:

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  prefix: "."
```

Multiple endpoints example:

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  url: "/open/"
  tags: [open]
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  url: "/admin/"
  basic_auth: true
  username: adminuser
  password: somepassword
  tags: [admin]
```

Disable Content-Type checks

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  content_type: ""
  prefix: "json"
```

Basic auth and SSL example:

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  ssl.enabled: true
  ssl.certificate: "/home/user/server.pem"
  ssl.key: "/home/user/server.key"
  ssl.verification_mode: "none"
  ssl.certificate_authority: "/home/user/ca.pem"
  basic_auth: true
  username: someuser
  password: somepassword
```

Authentication or checking that a specific header includes a specific value

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  secret.header: someheadername
  secret.value: secretheadertoken
```

Validate webhook endpoint for a specific provider using CRC

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  secret.header: someheadername
  secret.value: secretheadertoken
  crc.provider: webhookProvider
  crc.secret: secretToken
```

Validate a HMAC signature from a specific header

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  hmac.header: "X-Hub-Signature-256"
  hmac.key: "password123"
  hmac.type: "sha256"
  hmac.prefix: "sha256="
```

Preserving original event and including headers in document

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  preserve_original_event: true
  include_headers: ["TestHeader"]
```

Common Expression Language example:

```yaml
filebeat.inputs:
- type: http_endpoint
  enabled: true
  listen_address: 192.168.1.1
  listen_port: 8080
  program: |
    obj.records.map(r, {
     "requestId": obj.requestId,
     "timestamp": string(obj.timestamp),
     "event": r,
    })
```

This example would allow handling of a JSON body that is an object containing more than one event that each should be ingested as separate documents with the common timestamp and request ID:

```json
{
  "requestId": "ed4acda5-034f-9f42-bba1-f29aea6d7d8f",
  "timestamp": 1578090901599,
  "records": [
    {
      "data": "event record 1"
    },
    {
      "data": "event record 2"
    }
  ]
}
```

## Configuration options [_configuration_options_10]

The `http_endpoint` input supports the following configuration options plus the [Common options](#filebeat-input-http_endpoint-common-options) described later.


### `basic_auth` [_basic_auth]

Enables or disables HTTP basic auth for each incoming request. If enabled then `username` and `password` will also need to be configured.


### `username` [_username]

If `basic_auth` is enabled, this is the username used for authentication against the HTTP listener. Requires `password` to also be set.


### `password` [_password]

If `basic_auth` is enabled, this is the password used for authentication against the HTTP listener. Requires `username` to also be set.


### `secret.header` [_secret_header]

The header to check for a specific value specified by `secret.value`. Certain webhooks provide the possibility to include a special header and secret to identify the source.


### `secret.value` [_secret_value]

The secret stored in the header name specified by `secret.header`. Certain webhooks provide the possibility to include a special header and secret to identify the source.


### `hmac.header` [_hmac_header]

The name of the header that contains the HMAC signature: `X-Dropbox-Signature`, `X-Hub-Signature-256`, etc. HMAC signatures may be encoded as hex or base64.


### `hmac.key` [_hmac_key]

The secret key used to calculate the HMAC signature. Typically, the webhook sender provides this value.


### `hmac.type` [_hmac_type]

The hash algorithm to use for the HMAC comparison. At this time the only valid values are `sha256` or `sha1`.


### `hmac.prefix` [_hmac_prefix]

The prefix for the signature. Certain webhooks prefix the HMAC signature with a value, for example `sha256=`.


### `content_type` [_content_type]

By default the input expects the incoming POST to include a Content-Type of `application/json` to try to enforce the incoming data to be valid JSON. In certain scenarios when the source of the request is not able to do that, it can be overwritten with another value or set to null.


### `max_in_flight_bytes` [_max_in_flight_bytes]

The total sum of request body lengths that are allowed at any given time. If non-zero, the input will compare this value to the sum of in-flight request body lengths from requests that include a `wait_for_completion_timeout` request query and will return a 503 HTTP status code, along with a Retry-After header configured with the `retry_after` option. The default value for this option is zero, no limit.


### `retry_after` [_retry_after]

If a request has exceeded the `max_in_flight_bytes` limit, the response to the client will include a Retry-After header specifying how many seconds the client should wait to retry again. The default value for this option is 10 seconds.


### `program` [_program]

The normal operation of the input treats the body either as a single event when the body is an object, or as a set of events when the body is an array. If the body should be handled differently, for example a set of events in an array field of an object to be handled as a set of events, then a [Common Expression Language (CEL)](https://opensource.google.com/projects/cel) program can be provided through this configuration field. The name of the object in the CEL program is `obj`. No CEL extensions are provided beyond the function in the CEL [standard library](https://github.com/google/cel-spec/blob/master/doc/langdef.md#standard). CEL [optional types](https://pkg.go.dev/github.com/google/cel-go/cel#OptionalTypes) are supported.

Note that during evaluation, numbers that are not representable exactly within a double floating point value will be converted to a string to avoid data corruption.


### `response_code` [_response_code]

The HTTP response code returned upon success. Should be in the 2XX range.


### `response_body` [_response_body]

The response body returned upon success.


### `listen_address` [_listen_address]

If multiple interfaces is present the `listen_address` can be set to control which IP address the listener binds to. Defaults to `127.0.0.1`.


### `listen_port` [_listen_port]

Which port the listener binds to. Defaults to 8000.


### `url` [_url]

This options specific which URL path to accept requests on. Defaults to `/`


### `prefix` [_prefix]

This option specifies which prefix the incoming request will be mapped to. If `prefix` is "`.`", the request will be mapped to the root of the resulting document.


### `include_headers` [_include_headers]

This options specifies a list of HTTP headers that should be copied from the incoming request and included in the document. All configured headers will always be canonicalized to match the headers of the incoming request. For example, `["content-type"]` will become `["Content-Type"]` when the filebeat is running.


### `preserve_original_event` [_preserve_original_event]

This option includes the JSON representation of the incoming request in the `event.original` field as a string before sending the event to Elasticsearch. The representation may not be a verbatim copy of the original message, but is guaranteed to be an [RFC7493](https://datatracker.ietf.org/doc/html/rfc7493) compliant message.


### `crc.provider` [_crc_provider]

This option defines the provider of the webhook that uses CRC (Challenge-Response Check) for validating the endpoint. The HTTP endpoint input is responsible for ensuring the authenticity of incoming webhook requests by generating and verifying a unique token. By specifying the `crc.provider`, you ensure that the system correctly handles the specific CRC validation process required by the chosen provider.


### `crc.secret` [_crc_secret]

The secret token provided by the webhook owner for the CRC validation. It is required when a `crc.provider` is set.


### `method` [_method]

The HTTP method handled by the endpoint. If specified, `method` must be `POST`, `PUT` or `PATCH`. The default method is `POST`. If `PUT` or `PATCH` are specified, requests using those method types are accepted, but are treated as `POST` requests and are expected to have a request body containing the request data.


### `tracer.enabled` [_tracer_enabled_3]

It is possible to log HTTP requests to a local file-system for debugging configurations. This option is enabled by setting `tracer.enabled` to true and setting the `tracer.filename` value. Additional options are available to tune log rotation behavior. To delete existing logs, set `tracer.enabled` to false without unsetting the filename option.

Enabling this option compromises security and should only be used for debugging.


### `tracer.filename` [_tracer_filename_4]

To differentiate the trace files generated from different input instances, a placeholder `*` can be added to the filename and will be replaced with the input instance id. For Example, `http-request-trace-*.ndjson`.


### `tracer.maxsize` [_tracer_maxsize_2]

This value sets the maximum size, in megabytes, the log file will reach before it is rotated. By default logs are allowed to reach 1MB before rotation. Individual request bodies will be truncated to a maximum size of 10kiB.


### `tracer.maxage` [_tracer_maxage]

This specifies the number days to retain rotated log files. If it is not set, log files are retained indefinitely.


### `tracer.maxbackups` [_tracer_maxbackups]

The number of old logs to retain. If it is not set all old logs are retained subject to the `tracer.maxage` setting.


### `tracer.localtime` [_tracer_localtime]

Whether to use the host’s local time rather that UTC for timestamping rotated log file names.


### `tracer.compress` [_tracer_compress]

This determines whether rotated logs should be gzip compressed.


## Metrics [_metrics_11]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `bind_address` | Bind address of input. |
| `route` | HTTP request route of the input. |
| `is_tls_connection` | Whether the input is listening on a TLS connection. |
| `api_errors_total` | Number of API errors. |
| `batches_received_total` | Number of event arrays received. |
| `batches_published_total` | Number of event arrays published. |
| `batches_acked_total` | Number of event arrays ACKed. |
| `events_published_total` | Number of events published. |
| `size` | Histogram of request content lengths. |
| `batch_size` | Histogram of the received event array length. |
| `batch_processing_time` | Histogram of the elapsed successful batch processing times in nanoseconds (time of receipt to time of ACK for non-empty batches). |
| `batch_ack_time` | Histogram of the elapsed successful batch ACKing times in nanoseconds (time of handler start to time of ACK for non-empty batches). |


## Common options [filebeat-input-http_endpoint-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_12]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_12]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: http_endpoint
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-http_endpoint-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: http_endpoint
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-http_endpoint]

If this option is set to true, the custom [fields](#filebeat-input-http_endpoint-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_12]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_12]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_12]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_12]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_12]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


