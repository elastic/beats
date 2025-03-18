---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_step_1_set_an_identifier_for_each_filestream_input.html
---

# Step 1: Set an identifier for each filestream input [_step_1_set_an_identifier_for_each_filestream_input]

All `filestream` inputs require an ID. Ensure you set a unique identifier for every input.

::::{important}
Never change the ID of an input, or you will end up with duplicate events.
::::

:::{tip}
The [take over](filebeat-input-filestream-take-over) mode can be used
to migrate states from old `filestream` inputs with different IDs.
:::


```yaml
filebeat.inputs:
- type: filestream
  enabled: true
  id: my-java-collector
  paths:
    - /var/log/java-exceptions*.log

- type: filestream
  enabled: true
  id: my-application-input
  paths:
    - /var/log/my-application*.json

- type: filestream
  enabled: true
  id: my-old-files
  paths:
    - /var/log/my-old-files*.log
```

