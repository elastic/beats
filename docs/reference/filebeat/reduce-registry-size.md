---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/reduce-registry-size.html
---

# Registry file is too large [reduce-registry-size]

Filebeat keeps the state of each file and persists the state to disk in the registry file. The file state is used to continue file reading at a previous position when Filebeat is restarted. If a large number of new files are produced every day, the registry file might grow to be too large. To reduce the size of the registry file, there are two configuration options available: [`clean_removed`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-clean-removed) and [`clean_inactive`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-clean-inactive).

For old files that you no longer touch and are ignored (see [`ignore_older`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-ignore-older)), we recommended that you use `clean_inactive`. If old files get removed from disk, then use the `clean_removed` option.

