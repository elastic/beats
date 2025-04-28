---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-network-volumes.html
---

# Can't read log files from network volumes [filebeat-network-volumes]

We do not recommend reading log files from network volumes. Whenever possible, install Filebeat on the host machine and send the log files directly from there. Reading files from network volumes (especially on Windows) can have unexpected side effects. For example, changed file identifiers may result in Filebeat reading a log file from scratch again.

If it is not possible to read from the host, then using the [`fingerprint`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-file-identity-fingerprint) file identity is the next best option.

