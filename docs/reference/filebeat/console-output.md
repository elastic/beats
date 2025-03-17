---
navigation_title: "Console"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/console-output.html
---

# Configure the Console output [console-output]


The Console output writes events in JSON format to stdout.

::::{warning}
The Console output should be used only for debugging issues as it can produce a large amount of logging data.
::::


To use this output, edit the Filebeat configuration file to disable the {{es}} output by commenting it out, and enable the console output by adding `output.console`.

Example configuration:

```yaml
output.console:
  pretty: true
```

## Configuration options [_configuration_options_30]

You can specify the following `output.console` options in the `filebeat.yml` config file:

### `enabled` [_enabled_35]

The enabled config is a boolean setting to enable or disable the output. If set to false, the output is disabled.

The default value is `true`.


### `pretty` [_pretty]

If `pretty` is set to true, events written to stdout will be nicely formatted. The default is false.


### `codec` [_codec_4]

Output codec configuration. If the `codec` section is missing, events will be json encoded using the `pretty` option.

See [Change the output codec](/reference/filebeat/configuration-output-codec.md) for more information.


### `bulk_max_size` [_bulk_max_size_4]

The maximum number of events to buffer internally during publishing. The default is 2048.

Specifying a larger batch size may add some latency and buffering during publishing. However, for Console output, this setting does not affect how events are published.

Setting `bulk_max_size` to values less than or equal to 0 disables the splitting of batches. When splitting is disabled, the queue decides on the number of events to be contained in a batch.


### `queue` [_queue_6]

Configuration options for internal queue.

See [Internal queue](/reference/filebeat/configuring-internal-queue.md) for more information.

Note:`queue` options can be set under `filebeat.yml` or the `output` section but not both.



