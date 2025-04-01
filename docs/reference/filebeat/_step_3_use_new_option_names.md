---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_step_3_use_new_option_names.html
---

# Step 3: Use new option names [_step_3_use_new_option_names]

Several options are renamed in `filestream`. You can find a table with all of the changed configuration names at the end of this guide.

The most significant change you have to know about is in parsers. The configuration of `multiline`, `json`, and other parsers has changed. Now the ordering is configurable, so `filestream` expects a list of parsers. Furthermore, the `json` parser was renamed to `ndjson`.

The example configuration shown earlier needs to be adjusted as well:

```yaml
- type: filestream
  enabled: true
  id: my-java-collector
  take_over:
    enabled: true
  paths:
    - /var/log/java-exceptions*.log
  parsers:
    - multiline:
        pattern: '^\['
        negate: true
        match: after
  close.on_state_change.removed: true
  close.on_state_change.renamed: true

- type: filestream
  enabled: true
  id: my-application-input
  take_over:
    enabled: true
  paths:
    - /var/log/my-application*.json
  prospector.scanner.check_interval: 1m
  parsers:
    - ndjson:
        keys_under_root: true

- type: filestream
  enabled: true
  id: my-old-files
  take_over:
    enabled: true
  paths:
    - /var/log/my-old-files*.log
  ignore_inactive: since_last_start
```

|     |     |
| --- | --- |
| **Option name in log input** | **Option name in filestream input** |
| recursive_glob.enabled | prospector.scanner.recursive_glob |
| harvester_buffer_size | buffer_size |
| max_bytes | message_max_bytes |
| json | parsers.n.ndjson |
| multiline | parsers.n.multiline |
| exclude_files | prospector.scanner.exclude_files |
| close_inactive | close.on_state_change.inactive |
| close_removed | close.on_state_change.removed |
| close_eof | close.reader.on_eof |
| close_timeout | close.reader.after_interval |
| close_inactive | close.on_state_change.inactive |
| scan_frequency | prospector.scanner.check_interval |
| tail_files | ignore_inactive.since_last_start |
| symlinks | prospector.scanner.symlinks |
| backoff | backoff.init |
| backoff_max | backoff.max |

