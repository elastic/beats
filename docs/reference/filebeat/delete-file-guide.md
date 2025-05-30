# Removing files after ingestion

::::{warning}
Enabling this feature will remove files, which could lead to data loss.
::::

The Filestream input can remove files after they have been fully
ingested, three requirements need to be met before the Filestream
input can remove a file:
1. Conditions have been met for filestream to close the file. This is
   controlled by the close.* options.
2. Events from the file have been received by the configured output
   without error. (example elasticsearch output has indexed event or
   logstash has written event to persistent queue).
3. The `delete.grace_period` has expired and there is no new data on
   the file.

## How it works
Once a reader for a file is closed, either by reaching EOF (end of
file) or due to inactivity, Filestream will check if all events have
been published, if this is true, then it will wait for the configured
grace period, check if no new data has been added to the file, by
comparing its current size with the size when the last event was read,
then it will try to remove the file.

If any of the checks fail, the harvester is closed. One the next
file system scan happens, a new harvester, will be
started, once the close condition (EOF or inactivity) is met, then the
remove process will start again.

Once all checks are successful the file is removed.

## EOF x Inactivity
Filestream's reader can be configured to close on two conditions: EOF
and inactivity, each one has a different purpose:
 - EOF: it is recommended for files that do not have data appended to
   them, like a cronjob that when it is done copies the file to a
   folder where Filestream can read it;
 - Inactive: it is recommended for files that have data appended to
   them, like long running process that does not performs its own log
   rotation.
 
::::{note}
Even for short lived process that write their own log file within a
few seconds, avoid using EOF because Filestream might read until EOF
before the last entries are written to the file.
::::

## Examples
### Log files from old cronjobs
Filebeat will be used to ingest log files from old cronjobs, all files
have been fully written and Filebeat should remove them once it
finishes publishing all data. The log files are located at
`/var/log/cronjobs/*.log`.

For that the Filestream with delete on EOF will be used, the input
configuration is:
```yaml
  - type: filestream
    id: cronjobs-logs
    paths:
      - /var/log/cronjobs/*.log
    delete.on_close.eof: true
    delete.grace_period: 0s
```

### Log files from long running tasks
Filebeat will be used to collect logs from long-running tasks that
continuously add information to their log files. While these tasks are
active, new log entries appear in their respective files located at
`/var/log/long-tasks/*.log` every few seconds. Filebeat monitors these
files, and once a log file hasn't been updated for several minutes, it
indicates that the corresponding task has likely finished, making it
safe to remove the log file.

For this case Filestream can be configured to remove files after a
period of inactivity, the simplest configuration is:

```yaml
  - type: filestream
    id: long-tasks-logs
    paths:
      - /var/log/long-tasks/*.log
    delete.on_close.inactive: true
    delete.grace_period: 0s
```

Because `delete.on_close.inactive: true` the time to consider a file
inactive and close the reader is automatically set to 30 minutes. This
can be overwritten to a short or longer time, e.g: 5 minutes.

```yaml
  - type: filestream
    id: long-tasks-logs
    paths:
      - /var/log/long-tasks/*.log
    delete.on_close.inactive: true
    close.on_state_change.inactive: 5m
    delete.grace_period: 0s
```

### Waiting before removing log files
It is also possible to configure a grace period to wait after the
all events have been published before removing the file. Note that
this is different than the 'close on inactive' because the inactivity
timeout for the reader does not consider if an event has been
published, this means a file can be closed due to inactivity (no more
data read from it) even if some of its events are still in Filebeat's
publishing queue. In this example we want to remove files 5 minutes
after all events have been published and we know the files never have
data appended to them. For that we can use the EOF condition and
configure a grace period.

```yaml
  - type: filestream
    id: other-jobs
    paths:
      - /var/log/misc/*.log
    delete.on_close.eof: true
    delete.grace_period: 5m
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
file, the harvester will be closed after the grace period has passed,
then only on the next scan from the file system new data will be
picked up.
::::
