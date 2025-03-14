---
navigation_title: "Task scheduler"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/monitors-scheduler.html
---

# Configure the task scheduler [monitors-scheduler]


You specify options under `heartbeat.scheduler` to control the behavior of the task scheduler.

Example configuration:

```yaml
heartbeat.scheduler:
  limit: 10
  location: 'UTC-08:00'
```

In the example, setting `limit` to 10 guarantees that only 10 concurrent I/O tasks will be active. An I/O task can be the actual check or resolving an address via DNS.


## `limit` [heartbeat-scheduler-limit]

The number of concurrent I/O tasks that Heartbeat is allowed to execute. If set to 0, there is no limit. The default is 0.

Most operating systems set a file descriptor limit of 1024. For Heartbeat to operate correctly and not accidentally block libbeat output, the value that you specify for `limit` should be below the configured ulimit.


## `location` [heartbeat-scheduler-location]

The time zone for the scheduler. By default the scheduler uses localtime.


## `job.limit` [heartbeat-job-limit]

On top of the scheduler level limit, Heartbeat allows limiting the number of concurrent tasks per monitor/job type.

Example configuration:

```yaml
heartbeat.jobs:
  http:
    limit: 10
```

In the example, at any given time Heartbeat guarantees that only 10 concurrent `http` tasks will be active.

These limits can also be set via the environment variables `SYNTHETICS_LIMIT_{{TYPE}}`, where `{{TYPE}}` is one of `HTTP`, `TCP`, and `ICMP`.

