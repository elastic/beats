---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/windows-file-rotation.html
applies_to:
  stack: ga
  serverless: ga
---

# Open file handlers cause issues with Windows file rotation [windows-file-rotation]

On Windows, you might have problems renaming or removing files because Filebeat keeps the file handlers open. This can lead to issues with the file rotating system. To avoid this issue, you can use the [`close.on_state_change.removed`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-removed) and [`close.on_state_change.renamed`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-renamed) options together.

::::{important}
When you configure these options, files may be closed before the harvester has finished reading the files. If the file cannot be picked up again by the input and the harvester hasn’t finish reading the file, the missing lines will never be sent to the output.
::::
