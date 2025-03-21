---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/windows-file-rotation.html
---

# Open file handlers cause issues with Windows file rotation [windows-file-rotation]

On Windows, you might have problems renaming or removing files because Filebeat keeps the file handlers open. This can lead to issues with the file rotating system. To avoid this issue, you can use the [`close_removed`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-removed) and [`close_renamed`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-renamed) options together.

::::{important}
When you configure these options, files may be closed before the harvester has finished reading the files. If the file cannot be picked up again by the input and the harvester hasn’t finish reading the file, the missing lines will never be sent to the output.
::::


