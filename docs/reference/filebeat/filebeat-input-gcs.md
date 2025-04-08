---
navigation_title: "Google Cloud Storage"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-gcs.html
---

# Google Cloud Storage Input [filebeat-input-gcs]


Use the `google cloud storage input` to read content from files stored in buckets which reside on your Google Cloud. The input can be configured to work with and without polling, though if polling is disabled it will only perform a single collection of data, list the file contents and end the process.

**To mitigate errors and ensure a stable processing environment, this input employs the following features :**

1. When processing google cloud buckets, if suddenly there is any outage, the process will be able to resume post the last file it processed and was successfully able to save the state for.
2. If any errors occur for certain files, they will be logged appropriately, but the rest of the files will continue to be processed normally.
3. If any major error occurs which stops the main thread, the logs will be appropriately generated, describing said error.

**Config Option Removal Notice** : The `bucket_timeout` config option has been removed from the google cloud storage input. The intention behind this removal is to simplify the configuration and to make it more user friendly. The `bucket_timeout` option was confusing and had the potential to let users malconfigure the input, which could lead to unexpected behavior. The input now uses a more robust and efficient way to handle the bucket timeout internally.

::::{note}
:name: supported-types-gcs

Currently only `JSON` and `NDJSON` are supported object/file formats. Objects/files may be also be gzip compressed. "JSON credential keys" and "credential files" are supported authentication types. If an array is present as the root object for an object/file, it is automatically split into individual objects and processed. If a download for a file/object fails or gets interrupted, the download is retried for 2 times. This is currently not user configurable.
::::


$$$basic-config-gcs$$$
**A sample configuration with detailed explanation for each field is given below :-**

```yaml
filebeat.inputs:
- type: gcs
  id: my-gcs-id
  enabled: true
  project_id: my_project_id
  auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
  parse_json: true
  buckets:
  - name: gcs-test-new
    max_workers: 3
    poll: true
    poll_interval: 15s
  - name: gcs-test-old
    max_workers: 3
    poll: true
    poll_interval: 10s
```

**Explanation :** This `configuration` given above describes a basic gcs config having two buckets named `gcs-test-new` and `gcs-test-old`. Each of these buckets have their own attributes such as `name`, `max_workers`, `poll` and `poll_interval`. These attributes have detailed explanations given [below](#supported-attributes-gcs). For now lets try to understand how this config works.

For google cloud storage input to identify the files it needs to read and process, it will require the bucket names to be specified. We can have as many buckets as we deem fit. We are also able to configure the attributes `max_workers`, `poll` and `poll_interval` at the root level, which will then be applied to all buckets which do not specify any of these attributes explicitly.

::::{note}
If the attributes `max_workers`, `poll` and `poll_interval` are specified at the root level, these can still be overridden at the bucket level with different values, thus offering extensive flexibility and customization. Examples [below](#bucket-overrides) show this behavior.
::::


On receiving this config the google cloud storage input will connect to the service and retrieve a `Storage Client` using the given `bucket_name` and `auth.credentials_file`, then it will spawn two main go-routines, one for each bucket. After this each of these routines (threads) will initialize a scheduler which will in turn use the `max_workers` value to initialize an in-memory worker pool (thread pool) with `3` `workers` available. Basically that equates to two instances of a worker pool, one per bucket, each having 3 workers. These `workers` will be responsible for performing `jobs` that process a file (in this case read and output the contents of a file).

::::{note}
The scheduler is responsible for scheduling jobs, and uses the `maximum available workers` in the pool, at each iteration, to decide the number of files to retrieve and process. This keeps work distribution efficient. The scheduler uses `poll_interval` attribute value to decide how long to wait after each iteration. Each iteration consists of processing a certain number of files, decided by the `maximum available workers` value.
::::


**A Sample Response :-**

```json
{
  "@timestamp": "2022-09-01T13:54:24.588Z",
  "@metadata": {
    "beat": "filebeat",
    "type": "_doc",
    "version": "8.5.0",
    "_id": "gcs-test-new-data_3.json-worker-1"
  },
  "log": {
    "offset": 200,
    "file": {
      "path": "gs://gcs-test-new/data_3.json"
    }
  },
  "input": {
    "type": "gcs"
  },
  "message": "{\n    \"id\": 1,\n    \"title\": \"iPhone 9\",\n    \"description\": \"An apple mobile which is nothing like apple\",\n    \"price\": 549,\n    \"discountPercentage\": 12.96,\n    \"rating\": 4.69,\n    \"stock\": 94,\n    \"brand\": \"Apple\",\n    \"category\": \"smartphones\",\n    \"thumbnail\": \"https://dummyjson.com/image/i/products/1/thumbnail.jpg\",\n    \"images\": [\n        \"https://dummyjson.com/image/i/products/1/1.jpg\",\n        \"https://dummyjson.com/image/i/products/1/2.jpg\",\n        \"https://dummyjson.com/image/i/products/1/3.jpg\",\n        \"https://dummyjson.com/image/i/products/1/4.jpg\",\n        \"https://dummyjson.com/image/i/products/1/thumbnail.jpg\"\n    ]\n}\n",
  "cloud": {
    "provider": "goole cloud"
  },
  "gcs": {
    "storage": {
      "bucket": {
        "name": "gcs-test-new"
      },
      "object": {
        "name": "data_3.json",
        "content_type": "application/json",
        "json_data": [
          {
            "id": 1,
            "discountPercentage": 12.96,
            "rating": 4.69,
            "brand": "Apple",
            "price": 549,
            "category": "smartphones",
            "thumbnail": "https://dummyjson.com/image/i/products/1/thumbnail.jpg",
            "description": "An apple mobile which is nothing like apple",
            "title": "iPhone 9",
            "stock": 94,
            "images": [
              "https://dummyjson.com/image/i/products/1/1.jpg",
              "https://dummyjson.com/image/i/products/1/2.jpg",
              "https://dummyjson.com/image/i/products/1/3.jpg",
              "https://dummyjson.com/image/i/products/1/4.jpg",
              "https://dummyjson.com/image/i/products/1/thumbnail.jpg"
            ]
          }
        ]
      }
    }
  },
  "event": {
    "kind": "publish_data"
  }
}
```

As we can see from the response above, the `message` field contains the original stringified data while the `gcs.storage.object.data` contains the objectified data.

**Some of the key attributes are as follows :-**

1. **message** : Original stringified object data.
2. **log.file.path** : Path of the object in google cloud.
3. **gcs.storage.bucket.name** : Name of the bucket from which the file has been read.
4. **gcs.storage.object.name** : Name of the file/object which has been read.
5. **gcs.storage.object.content_type** : Content type of the file/object. You can find the supported content types [here](#supported-types-gcs) .
6. **gcs.storage.object.json_data** :  Objectified json file data, representing the contents of the file.

Now let’s explore the configuration attributes a bit more elaborately.

$$$supported-attributes-gcs$$$
**Supported Attributes :-**

1. [project_id](#attrib-project-id)
2. [auth.credentials_json.account_key](#attrib-auth-credentials-json)
3. [auth.credentials_file.path](#attrib-auth-credentials-file)
4. [buckets](#attrib-buckets)
5. [name](#attrib-bucket-name)
6. [max_workers](#attrib-max_workers-gcs)
7. [poll](#attrib-poll-gcs)
8. [poll_interval](#attrib-poll_interval-gcs)
9. [parse_json](#attrib-parse_json)
10. [file_selectors](#attrib-file_selectors-gcs)
11. [expand_event_list_from_field](#attrib-expand_event_list_from_field-gcs)
12. [timestamp_epoch](#attrib-timestamp_epoch-gcs)
13. [retry](#attrib-retry-gcs)


### `project_id` [attrib-project-id]

This attribute is required for various internal operations with respect to authentication, creating storage clients and logging which are used internally for various processing purposes.


### `auth.credentials_json.account_key` [attrib-auth-credentials-json]

This attribute contains the **json service account credentials string**, which can be generated from the google cloud console, ref: [https://cloud.google.com/iam/docs/creating-managing-service-account-keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys), under the respective storage account. A single storage account can contain multiple buckets, and they will all use this common service account access key.


### `auth.credentials_file.path` [attrib-auth-credentials-file]

This attribute contains the **service account credentials file**, which can be generated from the google cloud console, ref: [https://cloud.google.com/iam/docs/creating-managing-service-account-keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys), under the respective storage account. A single storage account can contain multiple buckets, and they will all use this common service account credentials file.

::::{note}
We require only either of `auth.credentials_json.account_key` or `auth.credentials_file.path` to be specified for authentication purposes. If both attributes are specified, then the one that occurs first in the configuration will be used.
::::



### `buckets` [attrib-buckets]

This attribute contains the details about a specific bucket like `name`, `max_workers`, `poll` and `poll_interval`. The attribute `name` is specific to a bucket as it describes the bucket name, while the fields `max_workers`, `poll` and `poll_interval` can exist both at the bucket level and the root level. This attribute is internally represented as an array, so we can add as many buckets as we require.


### `name` [attrib-bucket-name]

This is a specific subfield of a bucket. It specifies the bucket name.


### `max_workers` [attrib-max_workers-gcs]

This attribute defines the maximum number of workers (goroutines / lightweight threads) are allocated in the worker pool (thread pool) for processing jobs which read the contents of files. This attribute can be specified both at the root level of the configuration and at the bucket level. Bucket level values override the root level values if both are specified. Larger number of workers do not necessarily improve of throughput, and this should be carefully tuned based on the number of files, the size of the files being processed and resources available. Increasing `max_workers` to very high values may cause resource utilization problems and can lead to a bottleneck in processing. Usually a maximum cap of `2000` workers is recommended. A very low `max_worker` count will drastically increase the number of network calls required to fetch the objects, which can cause a bottleneck in processing.

::::{note}
The value of `max_workers` is tied to the `batch_size` currently to ensure even distribution of workloads across all goroutines. This ensures that the input is able to process the files in an efficient manner. This `batch_size` determines how many objects will be fetched in one single call. The `max_workers` value should be set based on the number of files to be read, the resources available and the network speed. For example,`max_workers=3` would mean that every pagination request a total number of `3` gcs objects are fetched and distributed among `3 goroutines`, `max_workers=100` would mean `100` gcs objects are fetched in every pagination request and distributed among `100 goroutines`.
::::



### `poll` [attrib-poll-gcs]

This attribute informs the scheduler whether to keep polling for new files or not. Default value of this is set to `true`. This attribute can be specified both at the root level of the configuration as well at the bucket level. The bucket level values will always take priority and override the root level values if both are specified.


### `poll_interval` [attrib-poll_interval-gcs]

This attribute defines the maximum amount of time after which the internal scheduler will make the polling call for the next set of objects/files. It can be defined in the following formats : `{{x}}s`, `{{x}}m`, `{{x}}h`, here `s = seconds`, `m = minutes` and `h = hours`. The value `{{x}}` can be anything we wish. Example : `10s` would mean we would like the polling to occur every 10 seconds. If no value is specified for this, by default its initialized to `5 minutes`. This attribute can be specified both at the root level of the configuration as well at the bucket level. The bucket level values will always take priority and override the root level values if both are specified. Having a lower `poll_interval` can make the input faster at the cost of more resource utilization.


### `parse_json` [attrib-parse_json]

This attribute informs the publisher  whether to parse & objectify json data or not. By default this is set to `false`, since it can get expensive dealing with highly nested json data. If this is set to `false` the **gcs.storage.object.json_data** field in the response will have an empty array. This attribute is only applicable for json objects and has no effect on other types of objects. This attribute can be specified both at the root level of the configuration as well at the bucket level. The bucket level values will always take priority and override the root level values if both are specified.


### `encoding` [input-gcs-encoding]

The file encoding to use for reading data that contains international characters. This only applies to non-JSON logs. See [`encoding`](/reference/filebeat/filebeat-input-log.md#_encoding_3).


### `decoding` [input-gcs-decoding]

The file decoding option is used to specify a codec that will be used to decode the file contents. This can apply to any file stream data. An example config is shown below:

Currently supported codecs are given below:-

1. [CSV](#attrib-decoding-csv-gcs): This codec decodes RFC 4180 CSV data streams.


### `the CSV codec` [attrib-decoding-csv-gcs]

The `CSV` codec is used to decode RFC 4180 CSV data streams. Enabling the codec without other options will use the default codec options.

```yaml
  decoding.codec.csv.enabled: true
```

The CSV codec supports five sub attributes to control aspects of CSV decoding. The `comma` attribute specifies the field separator character used by the CSV format. If it is not specified, the comma character *`,`* is used. The `comment` attribute specifies the character that should be interpreted as a comment mark. If it is specified, lines starting with the character will be ignored. Both `comma` and `comment` must be single characters. The `lazy_quotes` attribute controls how quoting in fields is handled. If `lazy_quotes` is true, a quote may appear in an unquoted field and a non-doubled quote may appear in a quoted field. The `trim_leading_space` attribute specifies that leading white space should be ignored, even if the `comma` character is white space. For complete details of the preceding configuration attribute behaviors, see the CSV decoder [documentation](https://pkg.go.dev/encoding/csv#Reader) The `fields_names` attribute can be used to specify the column names for the data. If it is absent, the field names are obtained from the first non-comment line of data. The number of fields must match the number of field names.

An example config is shown below:

```yaml
  decoding.codec.csv.enabled: true
  decoding.codec.csv.comma: "\t"
  decoding.codec.csv.comment: "#"
```


### `file_selectors` [attrib-file_selectors-gcs]

If the GCS buckets have objects that correspond to files that Filebeat shouldn’t process, `file_selectors` can be used to limit the files that are downloaded. This is a list of selectors which are based on a regular expression pattern. The regular expression should match the object name or should be a part of the object name (ideally a prefix). The regular expression syntax used is [RE2](https://github.com/google/re2/wiki/Syntax). Files that don’t match any configured expression won’t be processed.This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.

```yaml
filebeat.inputs:
- type: gcs
  project_id: my_project_id
  auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
  buckets:
  - name: obs-bucket
    max_workers: 3
    poll: true
    poll_interval: 15s
    file_selectors:
    - regex: '/Monitoring/'
    - regex: 'docs/'
    - regex: '/Security-Logs/'
```

The `file_selectors` operation is performed within the agent locally, hence using this option will cause the agent to download all the files and then filter them. This can cause a bottleneck in processing if the number of files is very high. It is recommended to use this attribute only when the number of files is limited or ample resources are available.


### `expand_event_list_from_field` [attrib-expand_event_list_from_field-gcs]

If the file-set using this input expects to receive multiple messages bundled under a specific field or an array of objects then the config option for `expand_event_list_from_field` can be specified. This setting will be able to split the messages under the group value into separate events. For example, if you have logs that are in JSON format and events are found under the JSON object "Records". To split the events into separate events, the config option `expand_event_list_from_field` can be set to "Records". This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.

```json
{
    "Records": [
        {
            "eventVersion": "1.07",
            "eventTime": "2019-11-14T00:51:00Z",
            "region": "us-east-1",
            "eventID": "EXAMPLE8-9621-4d00-b913-beca2EXAMPLE",
        },
        {
            "eventVersion": "1.07",
            "eventTime": "2019-11-14T00:52:00Z",
            "region": "us-east-1",
            "eventID": "EXAMPLEc-28be-486c-8928-49ce6EXAMPLE",
        }
    ]
}
```

```yaml
filebeat.inputs:
- type: gcs
  project_id: my_project_id
  auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
  buckets:
  - name: obs-bucket
    max_workers: 3
    poll: true
    poll_interval: 15s
    expand_event_list_from_field: Records
```

::::{note}
The `parse_json` setting is incompatible with `expand_event_list_from_field`. If enabled it will be ignored. This attribute is only applicable for JSON file formats. You do not need to specify this attribute if the file has an array of objects at the root level. Root level array of objects are automatically split into separate events. If failures occur or the input crashes due to some unexpected error, the processing will resume from the last successfully processed file or object.
::::



### `timestamp_epoch` [attrib-timestamp_epoch-gcs]

This attribute can be used to filter out files and objects that have a timestamp older than the specified value. The value of this attribute should be in unix `epoch` (seconds) format. The timestamp value is compared with the `object.Updated` field obtained from the object metadata. This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.

```yaml
filebeat.inputs:
- type: gcs
  project_id: my_project_id
  auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
  buckets:
  - name: obs-bucket
    max_workers: 3
    poll: true
    poll_interval: 15s
    timestamp_epoch: 1630444800
```

The GCS APIs don’t provide a direct way to filter files based on the timestamp, so the input will download all the files and then filter them based on the timestamp. This can cause a bottleneck in processing if the number of files are very high. It is recommended to use this attribute only when the number of files are limited or ample resources are available. This option scales vertically and not horizontally.


### `retry` [attrib-retry-gcs]

This attribute can be used to configure a list of sub attributes that directly control how the input should behave when a download for a file/object fails or gets interrupted.

* `max_attempts`: This attribute defines the maximum number of retry attempts(including the initial api call) that should be attempted for a retryable error. The default value for this is `3`.
* `initial_backoff_duration`: This attribute defines the initial backoff time. The default value for this is `1s`.
* `max_backoff_duration`: This attribute defines the maximum backoff time. The default value for this is `30s`.
* `backoff_multiplier`: This attribute defines the backoff multiplication factor. The default value for this is `2`.

::::{note}
The `initial_backoff_duration` and `max_backoff_duration` attributes must have time units. Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.
::::


By configuring these attributes, the user is given the flexibility to control how the input should behave when a download fails or gets interrupted. This attribute can only be specified at the root level of the configuration and not at the bucket level. It applies uniformly to all the buckets.

An example configuration is given below :-

```yaml
filebeat.inputs:
- type: gcs
  project_id: my_project_id
  auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
  retry:
    max_attempts: 3
    initial_backoff_duration: 2s
    max_backoff_duration: 60s
    backoff_multiplier: 2
  buckets:
  - name: obs-bucket
    max_workers: 3
    poll: true
    poll_interval: 11m
```

$$$bucket-overrides$$$
**The sample configs below will explain the bucket level overriding of attributes a bit further :-**

**CASE - 1 :**

Here `bucket_1` is using root level attributes while `bucket_2` overrides the values :

```yaml
filebeat.inputs:
- type: gcs
  id: my-gcs-id
  enabled: true
  project_id: my_project_id
  auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
  max_workers: 10
  poll: true
  poll_interval: 15s
  buckets:
  - name: bucket_1
  - name: bucket_2
    max_workers: 3
    poll: true
    poll_interval: 10s
```

**Explanation :** In this configuration `bucket_1` has no sub attributes in `max_workers`, `poll` and `poll_interval` defined. It inherits the values for these fileds from the root level, which is `max_workers = 10`, `poll = true` and `poll_interval = 15 seconds`. However `bucket_2` has these fields defined and it will use those values instead of using the root values.

**CASE - 2 :**

Here both `bucket_1` and `bucket_2` overrides the root values :

```yaml
filebeat.inputs:
  - type: gcs
    id: my-gcs-id
    enabled: true
    project_id: my_project_id
    auth.credentials_file.path: {{file_path}}/{{creds_file_name}}.json
    max_workers: 10
    poll: true
    poll_interval: 15s
    buckets:
    - name: bucket_1
      max_workers: 5
      poll: true
      poll_interval: 10s
    - name: bucket_2
      max_workers: 5
      poll: true
      poll_interval: 10s
```

**Explanation :** In this configuration even though we have specified `max_workers = 10`, `poll = true` and `poll_interval = 15s` at the root level, both the buckets will override these values with their own respective values which are defined as part of their sub attibutes.


## Metrics [_metrics_10]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `url` | URL of the input resource. |
| `errors_total` | Total number of errors encountered by the input. |
| `decode_errors_total` | Total number of decode errors encountered by the input. |
| `gcs_objects_requested_total` | Total number of GCS objects downloaded. |
| `gcs_objects_published_total` | Total number of GCS objects processed that were published. |
| `gcs_objects_listed_total` | Total number of GCS objects returned by list operations. |
| `gcs_bytes_processed_total` | Total number of GCS bytes processed. |
| `gcs_events_created_total` | Total number of events created from processing GCS data. |
| `gcs_failed_jobs_total` | Total number of failed jobs. |
| `gcs_expired_failed_jobs_total` | Total number of expired failed jobs that could not be recovered. |
| `gcs_objects_tracked_gauge` | Number of objects currently tracked in the state registry (gauge). |
| `gcs_objects_inflight_gauge` | Number of GCS objects inflight (gauge). |
| `gcs_jobs_scheduled_after_validation` | Histogram of the number of jobs scheduled after validation. |
| `gcs_object_processing_time` | Histogram of the elapsed GCS object processing times in nanoseconds (start of download to completion of parsing). |
| `gcs_object_size_in_bytes` | Histogram of processed GCS object size in bytes. |
| `gcs_events_per_object` | Histogram of event count per GCS object. |
| `source_lag_time` | Histogram of the time between the source (Updated) timestamp and the time the object was read, in nanoseconds. |

## Common input options [_common_input_options]



## Common options [filebeat-input-gcs-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_11]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_11]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: gcs
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-gcs-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: gcs
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-gcs]

If this option is set to true, the custom [fields](#filebeat-input-gcs-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_11]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_11]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_11]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_11]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_11]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.

::::{note}
Any feedback is welcome which will help us further optimize this input. Please feel free to open a github issue for any bugs or feature requests.
::::
