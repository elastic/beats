# Removing files after ingestion

::::{warning}
Enabling this feature will remove files, which could lead to unintentional data loss if not configured correctly.
::::

The Filestream input can remove files after they have been fully
ingested. Three requirements need to be met before the Filestream
input can remove a file:
1. Filestream has closed the file due to inactivity or because EOF has
   been reached. This is controlled by:
     - `close.on_state_change.inactive`
     - `close.reader.on_eof`
2. Events from the file have been received by the configured output
   without error. (example the Elasticsearch output has indexed all
   events or logstash has written event to persistent queue).
3. The `delete.grace_period` has expired and the file has not changed
   during the grace period.

## How it works
Once a reader for a file is closed, either by reaching EOF (end of
file) or due to inactivity, Filestream will check if all events have
been published. If this is true, then it will wait for the configured
grace period, check if no new data has been added to the file, by
comparing its current size with the size when the last event was read,
then it will try to remove the file. During the grace period Filebeat
monitors the file for changes and will close the harvester on any
change.

A published event is an event that has been acknowledged by the
output, an output always acknowledges a successfully written event,
however it will also acknowledge dropped events. Each output has
different conditions for dropping an event, refer the output's
documentation for more details.

If any of the checks fail, the harvester is closed. Once the next
file system scan happens, a new harvester will be
started, once the close condition (EOF or inactivity) is met, then the
remove process will start again.

Once all checks are successful the file is removed.

## EOF x Inactivity
Filestream's reader can be configured to close on two conditions: EOF
and inactivity, each one has a different purpose:
 - EOF: it is recommended for files that do not have data appended to
   them, like a cronjob that when it is done copies the file to a
   folder where Filestream can read it;
 - Inactivity: it is recommended for files that have data appended to
   them, like a long running process that does not performs its own log
   rotation.
 
::::{note}
When using close on EOF for files from short lived process that write
their logs within a few seconds, make sure to set an appropriate grace
period (default: 30 minutes) because even immutable copied files may
still "change" while being copied, especially across volumes or
network shares.
::::

## Examples
### Removing log files from old cronjobs
Filebeat will be used to ingest log files from old cronjobs, all files
have been fully written and Filebeat should remove them once it
finishes publishing all data. The log files are located at
`/var/log/cronjobs/*.log`. Once Filebeat finishes reading each file,
it will wait for 30min (the default), then delete them.

For that the Filestream with delete on EOF will be used, the input
configuration is:
```yaml
  - type: filestream
    id: cronjobs-logs
    paths:
      - /var/log/cronjobs/*.log
    close.reader.on_eof: true
    delete:
      enabled: true
```

#### Step-by-Step
1. Filebeat is configured with the above input and the Elasticsearch
   output.
2. Filebeat is started.
3. The Filestream input starts.
4. The prospector scans `/var/log/cronjobs/*.log` for files and finds
   all files.
5. A harvester is started for each file:
   1. The reader is started.
   2. The file is read until EOF.
   3. The reader closes because `close.reader.on_eof` is set to `true`.
   4. The harvester checks that all events have been published.
   5. If not all events have been published, the harvester is closed
      and it will be restarted in the next scan.
   6. If all events have been published, the grace period starts
      counting.
   7. If data is added to the file while waiting the grace period, the
      harvester is closed.
   8. Once the grace period expires, the file is checked once again
      for new data.
   9. If there was no change to the file, it is removed, otherwise the
      harvester is closed.

If Filebeat fails to remove the file, it will retry up to 5 times with
a constant backoff of 2 seconds. If all attempts fail, the harvester
is closed and a new harvester will be started in the next scan.

### Removing log files from long running tasks
Filebeat will be used to collect logs from long-running tasks that
continuously add information to their log files. While these tasks are
active, new log entries appear in their respective files located at
`/var/log/long-tasks/*.log` every few seconds. Filebeat monitors these
files, and once a log file hasn't been updated for several minutes, it
indicates that the corresponding task has likely finished, making it
safe to remove the log file. Once Filebeat closes the file, it will
wait for the grace period (30min by default), if the file has not
changed during the grace period, then the file is removed.

For this case Filestream can be configured to remove files after a
period of inactivity, the simplest configuration is:

```yaml
  - type: filestream
    id: long-tasks-logs
    paths:
      - /var/log/long-tasks/*.log
    close.on_state_change.inactive: 5m # That's the default, it can be omitted
    delete:
      enabled: true
```

### Waiting before removing log files
It is also possible to configure a grace period to wait after the
file has been closed and all events have been published before
removing the file. Note that this is different than the 'close on
inactive' because the inactivity timeout for the reader does not
consider if an event has been published, this means that a file can be
closed due to inactivity (no more data read from it) even if some of
its events are still in Filebeat's publishing queue. In this example
we want to remove files 5 minutes after all events have been published
and we know the files never have data appended to them. For that we
can use the EOF condition and configure a grace period.

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

The grace period will be counted after the harvester ensured all
events from the file have been published.

::::{tip}
Both `delete.grace_period` and `close.on_state_change.inactive` will
cause Filestream to wait after reading the last entry from the file,
however `close.on_state_change.inactive` will keep the reader open, so
new entries to the file can be quickly (almost in real time) picked
up, while `delete.grace_period` makes Filestream wait after the reader
has been closed and all events published, if new data is added to the
file, the harvester will be closed, then only on the next scan from
the file system new data will be picked up. While waiting for the
grace period to expire the harvesters checks the file for new data at
the same interval as the prospector, which is configured using (configured by
[`prospector.scanner.check_interval`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-scan-frequency)).
::::
