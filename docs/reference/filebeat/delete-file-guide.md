---
applies_to:
  stack: ga 9.2.0
---

# Removing files after ingestion

::::{warning}
This feature might lead to unintentional data loss if not configured correctly.
::::

The {{filestream}} input can remove files after they have been fully
ingested. {{filestream}} input can remove a file only when all these
conditions are met:
1. {{filestream}} has closed the file due to inactivity or because end-of-file (EOF) has
   been reached. This is controlled by:
     - `close.on_state_change.inactive`
     - `close.reader.on_eof`
2. Events from the file have been received by the output
   without error. For example, the Elasticsearch output has indexed all
   events or logstash has written event to persistent queue.
3. The `delete.grace_period` has expired and the file has not changed
   during the grace period.

## How it works
After a reader for a file is closed, either by reaching EOF or due
to inactivity, {{filestream}} checks if all events have been acknowledged.
If this is true, then {{filestream}} waits for the configured grace period,
checks if no new data has been added to the file by
comparing its current size to the size when the last event was read,
then tries to remove the file.

During the grace period, {{filebeat}} monitors the file. If the file size
changes, the grace period is interrupted and the file resumes
ingesting after the next file system scan.

An output always acknowledges a successfully written event,
however it also acknowledges dropped events. Each output has
different conditions for dropping an event. Refer the
[output's](/reference/filebeat/configuring-output.md) documentation
for more details.

If any of the checks fail, the harvester is closed. After the next
file system scan happens, a new harvester starts. If the close
condition (EOF or inactivity) is met, the remove process restarts.

After all checks are successful the file is removed.

## EOF and inactivity

{{filestream}}'s reader can be configured to close on two conditions: EOF
and inactivity. Each one has a different purpose:

 - EOF:  Recommended for files that don't have data appended to
   them, like a cronjob that copies the file to a folder.
 - Inactivity: Recommended for files that have data appended to
   them, like a long running process that does not rotate logs.

::::{note}
When using close on EOF for files from short lived processes that write
their logs within a few seconds, make sure to set an appropriate grace
period. Even immutable copied files might still change while being copied,
especially across volumes or network shares.
::::

## Examples
### Removing log files from old cronjobs
{{filebeat}} is ingesting log files from old cronjobs, all files
have been fully written and {{filebeat}} should remove them once it
finishes publishing all data. The log files are located at
`/var/log/cronjobs/*.log`. Once {{filebeat}} finishes reading each file,
it will wait for 30min (the default), then delete them.

For that close on EOF is used, the input configuration is:
```yaml
  - type: filestream
    id: cronjobs-logs
    paths:
      - /var/log/cronjobs/*.log
    close.reader.on_eof: true # Close the file on EOF
    delete:
      enabled: true
```

Here is a step by step of what happens within Filebeat when using the
previous configuration:

1. {{filebeat}} is configured with the above input and the Elasticsearch
   output.
2. {{filebeat}} is started.
3. The {{filestream}} input starts.
4. The prospector scans `/var/log/cronjobs/*.log` for files and finds
   all files.
5. A harvester is started for each file:
   1. The reader is started.
   2. The file is read until EOF.
   3. The reader closes because `close.reader.on_eof` is set to `true`.
   4. The harvester checks that all events have been acknowledged.
   5. If not all events have been acknowledged, the harvester is closed
      and it will be restarted in the next scan.
   6. If all events have been acknowledged, the grace period starts
      counting.
   7. If data is added to the file while waiting the grace period, the
      harvester is closed.
   8. Once the grace period expires, the file is checked once again
      for new data.
   9. If there was no change to the file, it is removed, otherwise the
      harvester is closed.

If {{filebeat}} fails to remove the file, it will retry up to 5 times with
a constant backoff of 2 seconds. If all attempts fail, the harvester
is closed and a new harvester will be started in the next scan.

### Removing log files from long running tasks

In this example, {{filebeat}} collects logs from long-running tasks that
continuously add information to their log files. While these tasks are
active, new log entries appear in their respective files located at
`/var/log/long-tasks/*.log` every few seconds. {{filestream}} monitors these
files, and when a log file hasn't been updated for several minutes, it
indicates that the corresponding task has likely finished, making it
safe to remove the log file. After {{filestream}} closes the file, it will
wait for the grace period (30 minutes by default): if the file has not
changed during the grace period, the file is removed.

This is the input configuration:

```yaml
  - type: filestream
    id: long-tasks-logs
    paths:
      - /var/log/long-tasks/*.log
    close.on_state_change.inactive: 5m # That's the default, it can be omitted.
    delete:
      enabled: true
```

### Waiting before removing log files

You can also configure a grace period to wait after the
file has been closed and all events have been acknowledged before
removing the file. This is different than the 'close on
inactive' because the inactivity timeout for the reader doesn't
consider if an event has been acknowledged. This means that a file can be
closed due to inactivity (no more data read from it) even if some of
its events are still in {{filebeat}}'s publishing queue.

In this example, files are removed 5 minutes after all events have been
acknowledged. We know the files never have data appended to them, so the
example uses close on EOF and configures a grace period.

```yaml
  - type: filestream
    id: other-jobs
    paths:
      - /var/log/misc/*.log
    close.reader.on_eof: true
    delete:
      enabled: true
      grace_period: 5m
```

The grace period is counted after the harvester ensured all
events from the file have been acknowledged.

::::{important}
Both `delete.grace_period` and `close.on_state_change.inactive` will
cause {{filestream}} to wait after reading the last entry from the file,
however `close.on_state_change.inactive` will keep the reader open, so
new entries to the file can be quickly (almost in real time) picked
up, while `delete.grace_period` makes {{filestream}} wait after the reader
has been closed and all events acknowledged, if new data is added to the
file, the harvester will be closed, then only on the next scan from
the file system new data will be picked up. While waiting for the
grace period to expire the harvesters checks the file for new data at
the same interval as the prospector, which is configured using
[`prospector.scanner.check_interval`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-scan-frequency).
::::
