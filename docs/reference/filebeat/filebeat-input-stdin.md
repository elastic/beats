---
navigation_title: "Stdin"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-stdin.html
---

# Stdin input [filebeat-input-stdin]


Use the `stdin` input to read events from standard in.

Note: This input cannot be run at the same time with other input types.

Example configuration:

```yaml
filebeat.inputs:
- type: stdin
```

## Configuration options [stdin-input-options]

The `stdin` input supports the following configuration options plus the [Common options](#filebeat-input-stdin-common-options) described later.


#### `encoding` [_encoding_4]

The file encoding to use for reading data that contains international characters. See the encoding names [recommended by the W3C for use in HTML5](http://www.w3.org/TR/encoding/).

Valid encodings:

* `plain`: plain ASCII encoding
* `utf-8` or `utf8`: UTF-8 encoding
* `gbk`: simplified Chinese charaters
* `iso8859-6e`: ISO8859-6E, Latin/Arabic
* `iso8859-6i`: ISO8859-6I, Latin/Arabic
* `iso8859-8e`: ISO8859-8E, Latin/Hebrew
* `iso8859-8i`: ISO8859-8I, Latin/Hebrew
* `iso8859-1`: ISO8859-1, Latin-1
* `iso8859-2`: ISO8859-2, Latin-2
* `iso8859-3`: ISO8859-3, Latin-3
* `iso8859-4`: ISO8859-4, Latin-4
* `iso8859-5`: ISO8859-5, Latin/Cyrillic
* `iso8859-6`: ISO8859-6, Latin/Arabic
* `iso8859-7`: ISO8859-7, Latin/Greek
* `iso8859-8`: ISO8859-8, Latin/Hebrew
* `iso8859-9`: ISO8859-9, Latin-5
* `iso8859-10`: ISO8859-10, Latin-6
* `iso8859-13`: ISO8859-13, Latin-7
* `iso8859-14`: ISO8859-14, Latin-8
* `iso8859-15`: ISO8859-15, Latin-9
* `iso8859-16`: ISO8859-16, Latin-10
* `cp437`: IBM CodePage 437
* `cp850`: IBM CodePage 850
* `cp852`: IBM CodePage 852
* `cp855`: IBM CodePage 855
* `cp858`: IBM CodePage 858
* `cp860`: IBM CodePage 860
* `cp862`: IBM CodePage 862
* `cp863`: IBM CodePage 863
* `cp865`: IBM CodePage 865
* `cp866`: IBM CodePage 866
* `ebcdic-037`: IBM CodePage 037
* `ebcdic-1040`: IBM CodePage 1140
* `ebcdic-1047`: IBM CodePage 1047
* `koi8r`: KOI8-R, Russian (Cyrillic)
* `koi8u`: KOI8-U, Ukranian (Cyrillic)
* `macintosh`: Macintosh encoding
* `macintosh-cyrillic`: Macintosh Cyrillic encoding
* `windows1250`: Windows1250, Central and Eastern European
* `windows1251`: Windows1251, Russian, Serbian (Cyrillic)
* `windows1252`: Windows1252, Legacy
* `windows1253`: Windows1253, Modern Greek
* `windows1254`: Windows1254, Turkish
* `windows1255`: Windows1255, Hebrew
* `windows1256`: Windows1256, Arabic
* `windows1257`: Windows1257, Estonian, Latvian, Lithuanian
* `windows1258`: Windows1258, Vietnamese
* `windows874`:  Windows874, ISO/IEC 8859-11, Latin/Thai
* `utf-16-bom`: UTF-16 with required BOM
* `utf-16be-bom`: big endian UTF-16 with required BOM
* `utf-16le-bom`: little endian UTF-16 with required BOM

The `plain` encoding is special, because it does not validate or transform any input.


#### `exclude_lines` [filebeat-input-stdin-exclude-lines]

A list of regular expressions to match the lines that you want Filebeat to exclude. Filebeat drops any lines that match a regular expression in the list. By default, no lines are dropped. Empty lines are ignored.

If [multiline](/reference/filebeat/multiline-examples.md#multiline) settings are also specified, each multiline message is combined into a single line before the lines are filtered by `exclude_lines`.

The following example configures Filebeat to drop any lines that start with `DBG`.

```yaml
filebeat.inputs:
- type: stdin
  ...
  exclude_lines: ['^DBG']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `include_lines` [filebeat-input-stdin-include-lines]

A list of regular expressions to match the lines that you want Filebeat to include. Filebeat exports only the lines that match a regular expression in the list. By default, all lines are exported. Empty lines are ignored.

If [multiline](/reference/filebeat/multiline-examples.md#multiline) settings also specified, each multiline message is combined into a single line before the lines are filtered by `include_lines`.

The following example configures Filebeat to export any lines that start with `ERR` or `WARN`:

```yaml
filebeat.inputs:
- type: stdin
  ...
  include_lines: ['^ERR', '^WARN']
```

::::{note}
If both `include_lines` and `exclude_lines` are defined, Filebeat executes `include_lines` first and then executes `exclude_lines`. The order in which the two options are defined doesn’t matter. The `include_lines` option will always be executed before the `exclude_lines` option, even if `exclude_lines` appears before `include_lines` in the config file.
::::


The following example exports all log lines that contain `sometext`, except for lines that begin with `DBG` (debug messages):

```yaml
filebeat.inputs:
- type: stdin
  ...
  include_lines: ['sometext']
  exclude_lines: ['^DBG']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `harvester_buffer_size` [_harvester_buffer_size_3]

The size in bytes of the buffer that each harvester uses when fetching a file. The default is 16384.


#### `max_bytes` [_max_bytes_3]

The maximum number of bytes that a single log message can have. All bytes after `max_bytes` are discarded and not sent. This setting is especially useful for multiline log messages, which can get large. The default is 10MB (10485760).


#### `json` [filebeat-input-stdin-config-json]

These options make it possible for Filebeat to decode logs structured as JSON messages. Filebeat processes the logs line by line, so the JSON decoding only works if there is one JSON object per line.

The decoding happens before line filtering and multiline. You can combine JSON decoding with filtering and multiline if you set the `message_key` option. This can be helpful in situations where the application logs are wrapped in JSON objects, as with like it happens for example with Docker.

Example configuration:

```yaml
json.keys_under_root: true
json.add_error_key: true
json.message_key: log
```

You must specify at least one of the following settings to enable JSON parsing mode:

**`keys_under_root`**
:   By default, the decoded JSON is placed under a "json" key in the output document. If you enable this setting, the keys are copied top level in the output document. The default is false.

**`overwrite_keys`**
:   If `keys_under_root` and this setting are enabled, then the values from the decoded JSON object overwrite the fields that Filebeat normally adds (type, source, offset, etc.) in case of conflicts.

**`expand_keys`**
:   If this setting is enabled, Filebeat will recursively de-dot keys in the decoded JSON, and expand them into a hierarchical object structure. For example, `{"a.b.c": 123}` would be expanded into `{"a":{"b":{"c":123}}}`. This setting should be enabled when the input is produced by an [ECS logger](https://github.com/elastic/ecs-logging).

**`add_error_key`**
:   If this setting is enabled, Filebeat adds a "error.message" and "error.type: json" key in case of JSON unmarshalling errors or when a `message_key` is defined in the configuration but cannot be used.

**`message_key`**
:   An optional configuration setting that specifies a JSON key on which to apply the line filtering and multiline settings. If specified the key must be at the top level in the JSON object and the value associated with the key must be a string, otherwise no filtering or multiline aggregation will occur.

**`document_id`**
:   Option configuration setting that specifies the JSON key to set the document id. If configured, the field will be removed from the original json document and stored in `@metadata._id`

**`ignore_decoding_error`**
:   An optional configuration setting that specifies if JSON decoding errors should be logged or not. If set to true, errors will not be logged. The default is false.


#### `multiline` [_multiline_7]

Options that control how Filebeat deals with log messages that span multiple lines. See [Multiline messages](/reference/filebeat/multiline-examples.md) for more information about configuring multiline options.


## Common options [filebeat-input-stdin-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_23]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_22]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: stdin
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-stdin-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: stdin
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-stdin]

If this option is set to true, the custom [fields](#filebeat-input-stdin-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_22]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_22]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_22]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_22]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_22]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


