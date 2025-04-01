---
navigation_title: "Azure Blob Storage"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-azure-blob-storage.html
---

# Azure Blob Storage Input [filebeat-input-azure-blob-storage]


Use the `azure blob storage input` to read content from files stored in containers which reside on your Azure Cloud. The input can be configured to work with and without polling, though currently, if polling is disabled it will only perform a one time passthrough, list the file contents and end the process. Polling is generally recommented for most cases even though it can get expensive with dealing with a very large number of files.

**To mitigate errors and ensure a stable processing environment, this input employs the following features :**

1. When processing azure blob containers, if suddenly there is any outage, the process will be able to resume post the last file it processed and was successfully able to save the state for.
2. If any errors occur for certain files, they will be logged appropriately, but the rest of the files will continue to be processed normally.
3. If any major error occurs which stops the main thread, the logs will be appropriately generated, describing said error.

::::{note}
:name: supported-types

`JSON`, `NDJSON` and `CSV` are supported blob/file formats. Blobs/files may be also be gzip compressed. `shared access keys`, `connection strings` and `Microsoft Entra ID RBAC` authentication types are supported.
::::


$$$basic-config$$$
**A sample configuration with detailed explanation for each field is given below :-**

```yaml
filebeat.inputs:
- type: azure-blob-storage
  id: my-azureblobstorage-id
  enabled: true
  account_name: some_account
  auth.shared_credentials.account_key: some_key
  containers:
  - name: container_1
    max_workers: 3
    poll: true
    poll_interval: 10s
  - name: container_2
    max_workers: 3
    poll: true
    poll_interval: 10s
```

**Explanation :** This `configuration` given above describes a basic blob storage config having two containers named `container_1` and `container_2`. Each of these containers have their own attributes such as `name`, `max_workers`, `poll` and `poll_interval`. These attributes have detailed explanations given [below](#supported-attributes). For now lets try to understand how this config works.

For azure blob storage input to identify the files it needs to read and process, it will require the container names to be specified. We can have as many containers as we deem fit. We are also able to configure the attributes `max_workers`, `poll` and `poll_interval` at the root level, which will then be applied to all containers which do not specify any of these attributes explicitly.

::::{note}
If the attributes `max_workers`, `poll` and `poll_interval` are specified at the root level, these can still be overridden at the container level with different values, thus offering extensive flexibility and customization. Examples [below](#container-overrides) show this behaviour.
::::


On receiving this config the azure blob storage input will connect to the service and retrieve a `ServiceClient` using the given `account_name` and `auth.shared_credentials.account_key`, then it will spawn two main go-routines, one for each container. After this each of these routines (threads) will initialize a scheduler which will in turn use the `max_workers` value to initialize an in-memory worker pool (thread pool) with `3` `workers` available. Basically that equates to two instances of a worker pool, one per container, each having 3 workers. These `workers` will be responsible if performing `jobs` that process a file (in this case read and output the contents of a file).

::::{note}
The scheduler is responsible for scheduling jobs, and uses the `maximum available workers` in the pool, at each iteration, to decide the number of files to retrieve and process. This keeps work distribution efficient. The scheduler uses `poll_interval` attribute value to decide how long to wait after each iteration. Each iteration consists of processing a certain number of files, decided by the `maximum available workers` value.
::::


**A Sample Response :-**

```json
 {
    "@timestamp": "2022-07-25T07:00:18.544Z",
    "@metadata": {
        "beat": "filebeat",
        "type": "_doc",
        "version": "8.4.0",
        "_id": "beatscontainer-data_3.json-worker-1"
    },
    "message": "{\n    \"id\": 3,\n    \"title\": \"Samsung Universe 9\",\n    \"description\": \"Samsung's new variant which goes beyond Galaxy to the Universe\",\n    \"price\": 1249,\n    \"discountPercentage\": 15.46,\n    \"rating\": 4.09,\n    \"stock\": 36,\n    \"brand\": \"Samsung\",\n    \"category\": \"smartphones\",\n    \"thumbnail\": \"https://dummyjson.com/image/i/products/3/thumbnail.jpg\",\n    \"images\": [\n        \"https://dummyjson.com/image/i/products/3/1.jpg\"\n    ]\n}",
    "cloud": {
        "provider": "azure"
    },
    "input": {
        "type": "azure-blob-storage"
    },
    "log": {
        "offset": 200,
        "file": {
            "path": "https://beatsblobstorage1.blob.core.windows.net/beatscontainer/data_3.json"
        }
    },
    "azure": {
        "storage": {
            "container": {
                "name": "beatscontainer"
            },
            "blob": {
                "content_type": "application/json",
                "name": "data_3.json"
            }
        }
    },
    "event": {
        "kind": "publish_data"
    }
}
```

As we can see from the response above, the `message` field contains the original stringified data.

**Some of the key attributes are as follows :-**

1. **message** : Original stringified blob data.
2. **log.file.path** : Path of the blob in azure cloud.
3. **azure.storage.blob.container.name** : Name of the container from which the file has been read.
4. **azure.storage.blob.object.name** : Name of the file/blob which has been read.
5. **azure.storage.blob.object.content_type** : Content type of the file/blob. You can find the supported content types [here](#supported-types).

Now let’s explore the configuration attributes a bit more elaborately.

$$$supported-attributes$$$
**Supported Attributes :-**

1. [account_name](#attrib-account-name)
2. [auth.oauth2](#attrib-auth-oauth2)
3. [auth.shared_credentials.account_key](#attrib-auth-shared-account-key)
4. [auth.connection_string.uri](#attrib-auth-connection-string)
5. [storage_url](#attrib-storage-url)
6. [containers](#attrib-containers)
7. [name](#attrib-container-name)
8. [max_workers](#attrib-max_workers)
9. [poll](#attrib-poll)
10. [poll_interval](#attrib-poll_interval)
11. [file_selectors](#attrib-file_selectors)
12. [expand_event_list_from_field](#attrib-expand_event_list_from_field)
13. [timestamp_epoch](#attrib-timestamp_epoch)


## `account_name` [attrib-account-name]

This attribute is required for various internal operations with respect to authentication, creating service clients and blob clients which are used internally for various processing purposes.


## `auth.oauth2` [attrib-auth-oauth2]

This attribute contains the Microsoft Entra ID RBAC authentication credentials for a secure connection to the Azure Blob Storage. The `auth.oauth2` attribute contains the following sub-attributes:

1. `client_id`: The client ID of the Azure Entra ID application.
2. `client_secret`: The client secret of the Azure Entra ID application.
3. `tenant_id`: The tenant ID of the Azure Entra ID application.

A sample configuration with `auth.oauth2` is given below:

```yaml
filebeat.inputs:
- type: azure-blob-storage
  account_name: some_account
  auth.oauth2:
    client_id: "some_client_id"
    client_secret: "some_client_secret"
    tenant_id: "some_tenant_id"
  containers:
  - name: container_1
    max_workers: 3
    poll: true
    poll_interval: 10s
```

How to setup the `auth.oauth2` credentials can be found in the Azure documentation [here](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-register-app)

::::{note}
According to our internal testing it seems that we require at least an access level of **blobOwner** for the service principle to be able to read the blobs. If you are facing any issues with the access level, ensure that the access level is set to **blobOwner**.
::::



## `auth.shared_credentials.account_key` [attrib-auth-shared-account-key]

This attribute contains the **access key**, found under the `Access keys` section on Azure Clound, under the respective storage account. A single storage account can contain multiple containers, and they will all use this common access key.


## `auth.connection_string.uri` [attrib-auth-connection-string]

This attribute contains the **connection string**, found under the `Access keys` section on Azure Clound, under the respective storage account. A single storage account can contain multiple containers, and they will all use this common connection string.

::::{note}
We require only either of `auth.shared_credentials.account_key` or `auth.connection_string.uri` to be specified for authentication purposes. If both attributes are specified, then the one that occurs first in the configuration will be used.
::::



## `storage_url` [attrib-storage-url]

Use this attribute to specify a custom storage URL if required. By default it points to azure cloud storage. Only use this if there is a specific need to connect to a different environment where blob storage is available.

**URL format :** `{{protocol}}://{{account_name}}.{{storage_uri}}`. This attribute resides at the root level of the config and not inside any container block.


## `containers` [attrib-containers]

This attribute contains the details about a specific container like `name`, `max_workers`, `poll` and `poll_interval`. The attribute `name` is specific to a container as it describes the container name, while the fields `max_workers`, `poll` and `poll_interval` can exist both at the container level and the root level. This attribute is internally represented as an array, so we can add as many containers as we require.


## `name` [attrib-container-name]

This is a specific subfield of a container. It specifies the container name.


## `max_workers` [attrib-max_workers]

This attribute defines the maximum number of workers allocated to the worker pool for processing jobs which read file contents. It can be specified both at the root level of the configuration, and at the container level. Container level values override root level values if both are specified. Larger number of workers do not necessarily improve throughput, and this should be carefully tuned based on the number of files, the size of the files being processed and resources available. Increasing `max_workers` to very high values may cause resource utilization problems and may lead to bottlenecks in processing. Usually a maximum of `2000` workers is recommended. A very low `max_worker` count will drastically increase the number of network calls required to fetch the blobs, which may cause a bottleneck in processing.

The batch size for workload distribution is calculated by the input to ensure that there is an even workload across all workers. This means that for a given `max_workers` parameter value, the input will calculate the optimal batch size for that setting. The `max_workers` value should be configured based on factors such as the total number of files to be processed, available system resources and network bandwidth.

Example:

* Setting `max_workers=3` would result in each request fetching `3 blobs` (batch size = 3), which are then distributed among `3 workers`.
* Setting `max_workers=100` would fetch `100 blobs` (batch size = 100) per request, distributed among `100 workers`.


## `poll` [attrib-poll]

This attribute informs the scheduler whether to keep polling for new files or not. Default value of this is `false`, so it will not keep polling if not explicitly specified. This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.


## `poll_interval` [attrib-poll_interval]

This attribute defines the maximum amount of time after which the internal scheduler will make the polling call for the next set of blobs/files. It can be defined in the following formats : `{{x}}s`, `{{x}}m`, `{{x}}h`, here `s = seconds`, `m = minutes` and `h = hours`. The value `{{x}}` can be anything we wish. Example : `10s` would mean we would like the polling to occur every 10 seconds. If no value is specified for this, by default its initialized to `300 seconds`. This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.


## `encoding` [input-azure-blob-storage-encoding]

The file encoding to use for reading data that contains international characters. This only applies to non-JSON logs. See [`encoding`](/reference/filebeat/filebeat-input-log.md#_encoding_3).


## `decoding` [input-azure-blob-storage-decoding]

The file decoding option is used to specify a codec that will be used to decode the file contents. This can apply to any file stream data. An example config is shown below:

Currently supported codecs are given below:-

1. [CSV](#attrib-decoding-csv-azureblobstorage): This codec decodes RFC 4180 CSV data streams.


## `the CSV codec` [attrib-decoding-csv-azureblobstorage]

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


## `file_selectors` [attrib-file_selectors]

If the Azure blob storage container will have blobs that correspond to files that Filebeat shouldn’t process, `file_selectors` can be used to limit the files that are downloaded. This is a list of selectors which are based on a `regex` pattern. The `regex` should match the blob name or should be a part of the blob name (ideally a prefix). The `regex` syntax is the same as used in the Go programming language. Files that don’t match any configured regex won’t be processed.This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.

```yaml
filebeat.inputs:
- type: azure-blob-storage
  id: my-azureblobstorage-id
  enabled: true
  account_name: some_account
  auth.shared_credentials.account_key: some_key
  containers:
  - name: container_1
    file_selectors:
    - regex: '/Monitoring/'
    - regex: 'docs/'
    - regex: '/Security-Logs/'
```

The `file_selectors` operation is performed within the agent locally. The agent will download all the files and then filter them based on the `file_selectors`. This can cause a bottleneck in processing if the number of files are very high. It is recommended to use this attribute only when the number of files are limited or ample resources are available.


## `expand_event_list_from_field` [attrib-expand_event_list_from_field]

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
- type: azure-blob-storage
  id: my-azureblobstorage-id
  enabled: true
  account_name: some_account
  auth.shared_credentials.account_key: some_key
  containers:
  - name: container_1
    expand_event_list_from_field: Records
```

::::{note}
This attribute is only applicable for JSON file formats. You do not require to specify this attribute if the file has an array of objects at the root level. Root level array of objects are automatically split into separate events. If failures occur or the input crashes due to some unexpected error, the processing will resume from the last successfully processed file/blob.
::::



## `timestamp_epoch` [attrib-timestamp_epoch]

This attribute can be used to filter out files/blobs which have a timestamp older than the specified value. The value of this attribute should be in unix `epoch` (seconds) format. The timestamp value is compared with the `LastModified Timestamp` obtained from the blob metadata. This attribute can be specified both at the root level of the configuration as well at the container level. The container level values will always take priority and override the root level values if both are specified.

```yaml
filebeat.inputs:
- type: azure-blob-storage
  id: my-azureblobstorage-id
  enabled: true
  account_name: some_account
  auth.shared_credentials.account_key: some_key
  containers:
  - name: container_1
    timestamp_epoch: 1627233600
```

The Azure Blob Storage APIs don’t provide a direct way to filter files based on timestamp, so the input will download all the files and then filter them based on the timestamp. This can cause a bottleneck in processing if the number of files are very high. It is recommended to use this attribute only when the number of files are limited or ample resources are available.

$$$container-overrides$$$
**The sample configs below will explain the container level overriding of attributes a bit further :-**

**CASE - 1 :**

Here `container_1` is using root level attributes while `container_2` overrides the values :

```yaml
filebeat.inputs:
- type: azure-blob-storage
  id: my-azureblobstorage-id
  enabled: true
  account_name: some_account
  auth.shared_credentials.account_key: some_key
  max_workers: 10
  poll: true
  poll_interval: 15s
  containers:
  - name: container_1
  - name: container_2
    max_workers: 3
    poll: true
    poll_interval: 10s
```

**Explanation :** In this configuration `container_1` has no sub attributes in `max_workers`, `poll` and `poll_interval` defined. It inherits the values for these fileds from the root level, which is `max_workers = 10`, `poll = true` and `poll_interval = 15 seconds`. However `container_2` has these fields defined and it will use those values instead of using the root values.

**CASE - 2 :**

Here both `container_1` and `container_2` overrides the root values :

```yaml
filebeat.inputs:
  - type: azure-blob-storage
    id: my-azureblobstorage-id
    enabled: true
    account_name: some_account
    auth.shared_credentials.account_key: some_key
    max_workers: 10
    poll: true
    poll_interval: 15s
    containers:
    - name: container_1
      max_workers: 5
      poll: true
      poll_interval: 10s
    - name: container_2
      max_workers: 5
      poll: true
      poll_interval: 10s
```

**Explanation :** In this configuration even though we have specified `max_workers = 10`, `poll = true` and `poll_interval = 15s` at the root level, both the containers will override these values with their own respective values which are defined as part of their sub attibutes.

::::{note}
Any feedback is welcome which will help us further optimize this input. Please feel free to open a github issue for any bugs or feature requests.
::::
