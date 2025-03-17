---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-cpu.html
---

# Filebeat is using too much CPU [filebeat-cpu]

Filebeat might be configured to scan for files too frequently. Check the setting for `scan_frequency` in the `filebeat.yml` config file. Setting `scan_frequency` to less than 1s may cause Filebeat to scan the disk in a tight loop.

