---
navigation_title: "Discard"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/discard-output.html
---

# Configure the Discard output [discard-output]


The Discard output throws away data.

::::{warning}
The Discard output should be used only for development or debugging issues. Data is lost.
::::


This can be useful if you want to work on your input configuration without needing to configure an output. It can also be useful to test how changes in input and processor configuration affect performance.

Example configuration:

```yaml
output.discard:
  enabled: true
```

## Configuration options [_configuration_options_8]

You can specify the following `output.discard` options in the `metricbeat.yml` config file:

### `enabled` [_enabled_8]

The enabled config is a boolean setting to enable or disable the output. If set to false, the output is disabled.

The default value is `true`.



