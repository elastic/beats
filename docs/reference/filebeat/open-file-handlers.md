---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/open-file-handlers.html
---

# Too many open file handlers [open-file-handlers]

Filebeat keeps the file handler open in case it reaches the end of a file so that it can read new log lines in near real time. If Filebeat is harvesting a large number of files, the number of open files can become an issue. In most environments, the number of files that are actively updated is low. The `close_inactive` configuration option should be set accordingly to close files that are no longer active.

There are additional configuration options that you can use to close file handlers, but all of them should be used carefully because they can have side effects. The options are:

* [`close_renamed`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-renamed)
* [`close_removed`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-removed)
* [`close_eof`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-eof)
* [`close_timeout`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-close-timeout)
* [`harvester_limit`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-harvester-limit)

The `close_renamed` and `close_removed` options can be useful on Windows to resolve issues related to file rotation. See [Open file handlers cause issues with Windows file rotation](/reference/filebeat/windows-file-rotation.md). The `close_eof` option can be useful in environments with a large number of files that have only very few entries. The `close_timeout` option is useful in environments where closing file handlers is more important than sending all log lines. For more details, see [Inputs](/reference/filebeat/configuration-filebeat-options.md).

Make sure that you read the documentation for these configuration options before using any of them.

