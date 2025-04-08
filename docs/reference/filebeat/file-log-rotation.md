---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/file-log-rotation.html
---

# Log rotation results in lost or duplicate events [file-log-rotation]

Filebeat supports reading from rotating log files. However, some log rotation strategies can result in lost or duplicate events when using Filebeat to forward messages. To resolve this issue:

* **Avoid log rotation strategies that copy and truncate log files**

    Log rotation strategies that copy and truncate the input log file can result in Filebeat sending duplicate events. This happens because Filebeat identifies files by inode and device name. During log rotation, lines that Filebeat has already processed are moved to a new file. When Filebeat encounters the new file, it reads from the beginning because the previous state information (the offset and read timestamp) is associated with the inode and device name of the old file.

    Furthermore, strategies that copy and truncate the input log file can result in lost events if lines are written to the log file after it’s copied, but before it’s truncated.

* **Make sure Filebeat is configured to read from all rotated logs**

    When an input log file is moved or renamed during log rotation, Filebeat is able to recognize that the file has already been read. After the file is rotated, a new log file is created, and the application continues logging. Filebeat picks up the new file during the next scan. Because the file has a new inode and device name, Filebeat starts reading it from the beginning.

    To avoid missing events from a rotated file, configure the input to read from the log file and all the rotated files. For examples, see [Example configurations](#log-rotate-example).


If you’re using Windows, also see [More about log rotation on Windows](#log-rotation-windows).


## Example configurations [log-rotate-example]

This section shows a typical configuration for logrotate, a popular tool for doing log rotation on Linux, followed by a Filebeat configuration that reads all the rotated logs.


### logrotate.conf [log-rotate-example-logrotate]

In this example, Filebeat reads web server log. The logs are rotated every day, and the new file is created with the specified permissions.

```yaml
/var/log/my-server/my-server.log {
    daily
    missingok
    rotate 7
    notifempty
    create 0640 www-data www-data
}
```


### filebeat.yml [log-rotate-example-filebeat]

In this example, Filebeat is configured to read all log files to make sure it does not miss any events.

```yaml
filebeat.inputs:
- type: filestream
  id: my-server-filestream-id
  paths:
  - /var/log/my-server/my-server.log*
```


## More about log rotation on Windows [log-rotation-windows]

On Windows, log rotation schemes that delete old files and rename newer files to old filenames might get blocked if the old files are being processed by Filebeat. This happens because Windows does not delete files and file metadata until the last process has closed the file. Unlike most *nix filesystems, a Windows filename cannot be reused until all processes accessing the file have closed the deleted file.

To avoid this problem, use dates in rotated filenames. The file will never be renamed to an older filename, and the log writer and log rotator will always be able to open the file. This approach also highly reduces the chance of log writing, rotation, and collection interfering with each other.

Because log rotation is typically handled by the logging application, we are not providing an example configuration for Windows.

Also read [Open file handlers cause issues with Windows file rotation](/reference/filebeat/windows-file-rotation.md).

