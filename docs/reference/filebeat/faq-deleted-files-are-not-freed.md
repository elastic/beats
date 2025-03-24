---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/faq-deleted-files-are-not-freed.html
---

# Filebeat keeps open file handlers of deleted files for a long time [faq-deleted-files-are-not-freed]

In the default behaviour, Filebeat opens the files and keeps them open until it reaches the end of them.  In situations when the configured output is blocked (e.g. {{es}} or {{ls}} is unavailable) for a long time, this can cause Filebeat to keep file handlers to files that were deleted from the file system in the mean time. As long as Filebeat keeps the deleted files open, the operating system doesn’t free up the space on disk, which can lead to increase disk utilisation or even out of disk situations.

To mitigate this issue, you can set the [`close_timeout`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-timeout) setting to `5m`. This will ensure every file handler is closed once every 5 minutes, regardless of whether it reached EOF or not. Note that this option can lead to data loss if the file is deleted before Filebeat reaches the end of the file.

