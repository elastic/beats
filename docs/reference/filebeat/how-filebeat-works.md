---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/how-filebeat-works.html
applies_to:
  stack: ga
---

# How Filebeat works [how-filebeat-works]

In this topic, you learn about the key building blocks of Filebeat and how they work together. Understanding these concepts will help you make informed decisions about configuring Filebeat for specific use cases.

Filebeat consists of two main components: [inputs](#input) and [harvesters](#harvester). These components work together to tail files and send event data to the output that you specify.


## What is a harvester? [harvester]

A harvester is responsible for reading the content of a single file. The harvester reads each file, line by line, and sends the content to the output. One harvester is started for each file. The harvester is responsible for opening and closing the file, which means that the file descriptor remains open while the harvester is running. If a file is removed or renamed while it’s being harvested, Filebeat continues to read the file. This has the side effect that the space on your disk is reserved until the harvester closes. By default, Filebeat keeps the file open until [`close.on_state_change.inactive`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-inactive) is reached.

Closing a harvester has the following consequences:

* The file handler is closed, freeing up the underlying resources if the file was deleted while the harvester was still reading the file.
* The harvesting of the file will only be started again after [`prospector.scanner.check_interval`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-scan-frequency) has elapsed.
* If the file is moved or removed while the harvester is closed, harvesting of the file will not continue.

To control when a harvester is closed, use the [`close_*`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-close-options) configuration options.


## What is an input? [input]

An input is responsible for managing the harvesters and finding all sources to read from.

If the input type is `filestream`, the input finds all files on the drive that match the defined glob paths and starts a harvester for each file. Each input runs in its own Go routine.

The following example configures Filebeat to harvest lines from all log files that match the specified glob patterns:

```yaml
filebeat.inputs:
- type: filestream
  id: unique-ID
  paths:
    - /var/log/*.log
    - /var/path2/*.log
```

Filebeat currently supports [several `input` types](/reference/filebeat/configuration-filebeat-options.md#filebeat-input-types). Each input type can be defined multiple times. The `filestream` input checks each file to see whether a harvester needs to be started, whether one is already running, or whether the file can be ignored (see [`ignore_older`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-ignore-older)). New lines are only picked up if the size of the file has changed since the harvester was closed.


## How does Filebeat keep the state of files? [_how_does_filebeat_keep_the_state_of_files]

Filebeat keeps the state of each file and frequently flushes the state to disk in the registry file. The state is used to remember the last offset a harvester was reading from and to ensure all log lines are sent. If the output, such as Elasticsearch or Logstash, is not reachable, Filebeat keeps track of the last lines sent and will continue reading the files as soon as the output becomes available again. While Filebeat is running, the state information is also kept in memory for each input. When Filebeat is restarted, data from the registry file is used to rebuild the state, and Filebeat continues each harvester at the last known position.

For each input, Filebeat keeps a state of each file it finds. Because files can be renamed or moved, the filename and path are not enough to identify a file. For each file, Filebeat stores unique identifiers to detect whether a file was harvested previously.

If your use case involves creating a large number of new files every day, you might find that the registry file grows to be too large. See [Registry file is too large](/reference/filebeat/reduce-registry-size.md) for details about configuration options that you can set to resolve this issue.


## How does Filebeat ensure at-least-once delivery? [at-least-once-delivery]

Filebeat guarantees that events will be delivered to the configured output at least once and with no data loss. Filebeat is able to achieve this behavior because it stores the delivery state of each event in the registry file.

In situations where the defined output is blocked and has not confirmed all events, Filebeat will keep trying to send events until the output acknowledges that it has received the events.

If Filebeat shuts down while it’s in the process of sending events, it does not wait for the output to acknowledge all events before shutting down. Any events that are sent to the output, but not acknowledged before Filebeat shuts down, are sent again when Filebeat is restarted. This ensures that each event is sent at least once, but you can end up with duplicate events being sent to the output. You can configure Filebeat to wait a specific amount of time before shutting down by setting the [`shutdown_timeout`](/reference/filebeat/configuration-general-options.md#shutdown-timeout) option.

::::{note}
There is a limitation to Filebeat’s at-least-once delivery guarantee involving log rotation and the deletion of old files. If log files are written to disk and rotated faster than they can be processed by Filebeat, or if files are deleted while the output is unavailable, data might be lost. On Linux, it’s also possible for Filebeat to skip lines as the result of inode reuse. See [*Common problems*](/reference/filebeat/faq.md) for more details about the inode reuse issue.
::::


