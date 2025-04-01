---
navigation_title: "Processors"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filtering-and-enhancing-data.html
---

# Filter and enhance data with processors [filtering-and-enhancing-data]


Your use case might require only a subset of the data exported by Filebeat, or you might need to enhance the exported data (for example, by adding metadata). Filebeat provides a couple of options for filtering and enhancing exported data.

You can configure each input to include or exclude specific lines or files. This allows you to specify different filtering criteria for each input. To do this, you use the `include_lines`, `exclude_lines`, and `exclude_files` options under the `filebeat.inputs` section of the config file (see [Inputs](/reference/filebeat/configuration-filebeat-options.md)). The disadvantage of this approach is that you need to implement a configuration option for each filtering criteria that you need.

Another approach (the one described here) is to define processors to configure global processing across all data exported by Filebeat.


## Processors [using-processors]

You can [define processors](/reference/filebeat/defining-processors.md) in your configuration to process events before they are sent to the configured output. The libbeat library provides processors for:

* reducing the number of exported fields
* enhancing events with additional metadata
* performing additional processing and decoding

Each processor receives an event, applies a defined action to the event, and returns the event. If you define a list of processors, they are executed in the order they are defined in the Filebeat configuration file.

```yaml
event -> processor 1 -> event1 -> processor 2 -> event2 ...
```

::::{important}
It’s recommended to do all drop and renaming of existing fields as the last step in a processor configuration. This is because dropping or renaming fields can remove data necessary for the next processor in the chain, for example dropping the `source.ip` field would remove one of the fields necessary for the `community_id` processor to function. If it’s necessary to remove, rename or overwrite an existing event field, please make sure it’s done by a corresponding processor ([`drop_fields`](/reference/filebeat/drop-fields.md), [`rename`](/reference/filebeat/rename-fields.md) or [`add_fields`](/reference/filebeat/add-fields.md)) placed at the end of the processor list defined in the input configuration.
::::



### Drop event example [drop-event-example]

The following configuration drops all the DEBUG messages.

```yaml
processors:
  - drop_event:
      when:
        regexp:
          message: "^DBG:"
```

To drop all the log messages coming from a certain log file:

```yaml
processors:
  - drop_event:
      when:
        contains:
          source: "test"
```


### Decode JSON example [decode-json-example]

In the following example, the fields exported by Filebeat include a field, `inner`, whose value is a JSON object encoded as a string:

```json
{ "outer": "value", "inner": "{\"data\": \"value\"}" }
```

The following configuration decodes the inner JSON object:

```yaml
filebeat.inputs:
- type: filestream
  paths:
    - input.json
  parsers:
    - ndjson:
        target: ""

processors:
  - decode_json_fields:
      fields: ["inner"]

output.console.pretty: true
```

The resulting output looks something like this:

```json subs=true
{
  "@timestamp": "2016-12-06T17:38:11.541Z",
  "beat": {
    "hostname": "host.example.com",
    "name": "host.example.com",
    "version": "{{stack-version}}"
  },
  "inner": {
    "data": "value"
  },
  "input": {
    "type": "log",
  },
  "offset": 55,
  "outer": "value",
  "source": "input.json",
  "type": "log"
}
```


















































