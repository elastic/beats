---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/inode-reuse-issue.html
---

# Inode reuse causes Filebeat to skip lines [inode-reuse-issue]

On Linux file systems, Filebeat uses the inode and device to identify files. When a file is removed from disk, the inode may be assigned to a new file. In use cases involving file rotation, if an old file is removed and a new one is created immediately afterwards, the new file may have the exact same inode as the file that was removed. In this case, Filebeat assumes that the new file is the same as the old and tries to continue reading at the old position, which is not correct.

By default states are never removed from the registry file. To resolve the inode reuse issue, we recommend that you use the [`clean_*`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-clean-options) options, especially [`clean_inactive`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-clean-inactive), to remove the state of inactive files. For example, if your files get rotated every 24 hours, and the rotated files are not updated anymore, you can set [`ignore_older`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-ignore-older) to 48 hours and [`clean_inactive`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-clean-inactive) to 72 hours.

You can use [`clean_removed`](/reference/filebeat/filebeat-input-log.md#filebeat-input-log-clean-removed) for files that are removed from disk. Be aware that `clean_removed` cleans the file state from the registry whenever a file cannot be found during a scan. If the file shows up again later, it will be sent again from scratch.

Aside from that you should also change the [`file_identity`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-file-identity) to [`fingerprint`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-file-identity-fingerprint). If you were using `native` (the default) or `path`, the state of the files will be automatically migrated to `fingerprint`.

