---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_step_2_enable_the_take_over_mode.html
---

# Step 2: Enable the take over mode [_step_2_enable_the_take_over_mode]

Now, to indicate that the new `filestream` is supposed to take over the files from a previously defined `log` input, we need to add `take_over: true` to each new `filestream`. This will make sure that the new `filestream` inputs will continue ingesting files from the same offset where the `log` inputs stopped.

::::{note}
It’s recommended to enable debug-level logs for Filebeat in order to follow the migration process. After the first run with `take_over: true` the setting can be removed.
::::


::::{warning}
The `take over` mode is in beta.
::::


::::{important}
If this parameter is not set, all the files will be re-ingested from the beginning and this will lead to data duplication. Please, double-check that this parameter is set.
::::


```yaml
logging:
  level: debug
filebeat.inputs:
- type: filestream
  enabled: true
  id: my-java-collector
  take_over: true
  paths:
    - /var/log/java-exceptions*.log

- type: filestream
  enabled: true
  id: my-application-input
  take_over: true
  paths:
    - /var/log/my-application*.json

- type: filestream
  enabled: true
  id: my-old-files
  take_over: true
  paths:
    - /var/log/my-old-files*.log
```

