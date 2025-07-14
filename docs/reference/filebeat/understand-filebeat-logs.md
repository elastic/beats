---
navigation_title: "Understand logged metrics"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/understand-filebeat-logs.html
---

# Understand metrics in Filebeat logs [understand-filebeat-logs]


Every 30 seconds (by default), Filebeat collects a snapshot of its internal metrics. It compares this snapshot with the previous one to identify metrics that have changed.

If any metric values have changed, Filebeat logs a summary. This summary includes two types of metrics:

* **Counters**: For metrics that count events (like `events.added` or `output.bytes_written`), the log shows the *change* since the last report. For example, if 100 events were processed since the last snapshot, the log will show that change.
* **Gauges**: For metrics that represent a point-in-time value (like `running_prospectors`), the log shows the *current* value.

This log entry is serialized as JSON and emitted at the `INFO` log level. Here is an example of such a log entry:

```json
{"log.level":"info","@timestamp":"2023-07-14T12:50:36.811Z","log.logger":"monitoring","log.origin":{"file.name":"log/log.go","file.line":187},"message":"Non-zero metrics in the last 30s","service.name":"filebeat","monitoring":{"metrics":{"beat":{"cgroup":{"memory":{"mem":{"usage":{"bytes":0}}}},"cpu":{"system":{"ticks":692690,"time":{"ms":60}},"total":{"ticks":3167250,"time":{"ms":150},"value":3167250},"user":{"ticks":2474560,"time":{"ms":90}}},"handles":{"limit":{"hard":1048576,"soft":1048576},"open":32},"info":{"ephemeral_id":"2bab8688-34c0-4522-80af-db86948d547d","uptime":{"ms":617670096},"version":"8.6.2"},"memstats":{"gc_next":57189272,"memory_alloc":43589824,"memory_total":275281335792,"rss":183574528},"runtime":{"goroutines":212}},"filebeat":{"events":{"active":5,"added":52,"done":49},"harvester":{"open_files":6,"running":6,"started":1}},"libbeat":{"config":{"module":{"running":15}},"output":{"events":{"acked":48,"active":0,"batches":6,"total":48},"read":{"bytes":210},"write":{"bytes":26923}},"pipeline":{"clients":15,"events":{"active":5,"filtered":1,"published":51,"total":52},"queue":{"max_events":3500,"filled":{"events":5,"bytes":6425,"pct":0.0014},"added":{"events":52,"bytes":65702},"consumed":{"events":52,"bytes":65702},"removed":{"events":48,"bytes":59277},"acked":48}}},"registrar":{"states":{"current":14,"update":49},"writes":{"success":6,"total":6}},"system":{"load":{"1":0.91,"15":0.37,"5":0.4,"norm":{"1":0.1138,"15":0.0463,"5":0.05}}}},"ecs.version":"1.6.0"}}
```


## Details [_details_2]

Focussing on the `.monitoring.metrics` field, and formatting the JSON, it’s value is:

```json
{
  "beat": {
    "cgroup": {
      "memory": {
        "mem": {"usage": {"bytes": 0}}
      }
    },
    "cpu": {
      "system": {"ticks":  692690, "time": {"ms":  60}                  },
      "total" : {"ticks": 3167250, "time": {"ms": 150}, "value": 3167250},
      "user"  : {"ticks": 2474560, "time": {"ms":  90}                  }
    },
    "handles": { "limit": {"hard": 1048576, "soft": 1048576}, "open": 32},
    "info": {
      "ephemeral_id": "2bab8688-34c0-4522-80af-db86948d547d",
      "uptime": {"ms": 617670096},
      "version": "8.6.2"
    },
    "memstats": {
      "gc_next"     :     57189272,
      "memory_alloc":     43589824,
      "memory_total": 275281335792,
      "rss"         :    183574528
    },
    "runtime": {"goroutines": 212}
  },
  "filebeat": {
    "events"   : {"active": 5, "added": 52, "done": 49},
    "harvester": {"open_files": 6, "running": 6, "started": 1}
  },
  "libbeat": {
    "config": {"module": {"running": 15} },
    "output": {
      "events": {"acked": 48, "active": 0, "batches": 6, "total": 48},
      "read"  : {"bytes": 210},
      "write" : {"bytes": 26923}
    },
    "pipeline": {
      "clients": 15,
      "events": {"active": 5, "filtered": 1, "published": 51, "total": 52},
      "queue": {
        "max_events": 3500,
        "filled":   {"events": 5,  "bytes": 6425, "pct": 0.0014},
        "added":    {"events": 52, "bytes": 65702},
        "consumed": {"events": 52, "bytes": 65702},
        "removed":  {"events": 48, "bytes": 59277},
        "acked": 48
      }
    }
  },
  "registrar": {
    "states": {"current": 14, "update": 49},
    "writes": {"success": 6, "total": 6}
  },
  "system": {
    "load": {
      "1":  0.91,
      "15": 0.37,
      "5":  0.4,
      "norm": {"1": 0.1138, "15": 0.0463, "5": 0.05}
    }
  }
}
```

The following tables explain the meaning of the most important fields under `.monitoring.metrics` and also provide hints that might be helpful in troubleshooting Filebeat issues.

| Field path (relative to `.monitoring.metrics`) | Type | Meaning |
| --- | --- | --- |
| `.beat` | Object | Information that is common to all Beats, e.g. version, goroutines, file handles, CPU, memory  |
| `.libbeat` | Object | Information about the publisher pipeline and output, also common to all Beats  |
| `.filebeat` | Object | Information specific to {{filebeat}}, e.g. harvester, events  |

| Field path (relative to `.monitoring.metrics.beat`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.cpu.system.time.ms` | Integer | CPU time spent in kernel mode, in milliseconds. | |
| `.cpu.user.time.ms` | Integer | CPU time spent in user mode, in milliseconds. | |
| `.cpu.total.time.ms` | Integer | Total CPU time (`system` + `user`), in milliseconds. | |
| `.cpu.ticks` | Integer | The arbitrary unit of time reported by the OS. The `time.ms` values are calculated from the difference in tick values between reports. | |
| `.memstats.gc_next` | Integer | The target heap size for the next garbage collection cycle, in bytes. | |
| `.memstats.memory_alloc` | Integer | Bytes of allocated heap objects. | A constantly growing value may indicate a memory leak. |
| `.memstats.memory_total` | Integer | Cumulative bytes allocated for heap objects. | A sustained, high rate of increase can indicate a memory leak or high memory churn. |
| `.memstats.rss` | Integer | Resident Set Size: total memory allocated to the process and held in RAM. | High RSS might indicate memory pressure on the system. |
| `.runtime.goroutines` | Integer | Number of goroutines running | If this number grows over time, it indicates a goroutine leak |

| Field path (relative to `.monitoring.metrics.system`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.load.1` | Float | System load average over the past 1 minute. | High values may indicate system-wide CPU pressure. |
| `.load.5` | Float | System load average over the past 5 minutes. | Compare with `.load.1` to see if load is sustained. If `.load.5` is significantly higher than `.load.1`, it may indicate a recent spike in load. |
| `.load.15` | Float | System load average over the past 15 minutes. | A high value here indicates a long-term load problem. If `.load.15` is significantly higher than `.load.5`, it may indicate a persistent load issue. |
| `.load.norm.1` | Float | Normalized system load average over the past 1 minute (divided by the number of CPUs). | Values consistently > 1.0 may indicate the system is overloaded. |
| `.load.norm.5` | Float | Normalized system load average over the past 5 minutes. | Compare with `.load.norm.1` to see the load trend. If `.load.norm.5` is consistently higher than `.load.norm.1`, it may indicate a growing load issue. |
| `.load.norm.15` | Float | Normalized system load average over the past 15 minutes. | A value consistently > 1.0 here strongly indicates the system is overloaded. |

| Field path (relative to `.monitoring.metrics.libbeat`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.pipeline.events.active` | Integer | Number of events currently in the libbeat publisher pipeline. | If this number grows over time, it may indicate that Filebeat is producing events faster than the output can consume them. Consider increasing the number of output workers (if this setting is supported by the output; {{es}} and {{ls}} outputs support this setting). The pipeline includes events currently being processed as well as events in the queue. So this metric can sometimes end up slightly higher than the queue size. If this metric reaches the maximum queue size (`queue.mem.events` for the in-memory queue), it almost certainly indicates backpressure on Filebeat, implying that Filebeat may temporarily stop ingesting events from the source until this backpressure is relieved. |
| `.output.events.total` | Integer | Number of events currently being processed by the output. | If this number grows over time, it may indicate that the output destination (e.g. {{ls}} pipeline or {{es}} cluster) is not able to accept events at the same or faster rate than what Filebeat is sending to it. |
| `.output.events.acked` | Integer | Number of events acknowledged by the output destination. | Generally, we want this number to be the same as `.output.events.total` as this indicates that the output destination has reliably received all the events sent to it. |
| `.output.events.failed` | Integer | Number of events that Filebeat tried to send to the output destination, but the destination failed to receive them. | Generally, we want this field to be absent or its value to be zero. When the value is greater than zero, it’s useful to check Filebeat’s logs right before this log entry’s `@timestamp` to see if there are any connectivity issues with the output destination. Note that failed events are not lost or dropped; they will be sent back to the publisher pipeline for retrying later. |
| `.output.events.dropped` | Integer | Number of events that Filebeat gave up sending to the output destination because of a permanent (non-retryable) error. |
| `.output.events.dead_letter` | Integer | Number of events that Filebeat successfully sent to a configured dead letter index after they failed to ingest in the primary index. |
| `.output.write.latency` | Object  | Reports statistics on the time to send an event to the connected output, in milliseconds. This can be used to diagnose delays and performance issues caused by I/O or output configuration. This metric is available for the Elasticsearch, file, Redis, and Logstash outputs. | These latency statistics are calculated over the lifetime of the connection. For long-lived connections, the average value will stabilize, making it less sensitive to short-term disruptions. |

| Field path (relative to `.monitoring.metrics.libbeat.pipeline`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.queue.max_events` | Integer (gauge) | The queue's maximum event count if it has one, otherwise zero. | |
| `.queue.max_bytes` | Integer (gauge) | The queue's maximum byte count if it has one, otherwise zero. | |
| `.queue.filled.events` | Integer (gauge) | Number of events currently stored by the queue. | |
| `.queue.filled.bytes` | Integer (gauge) | Number of bytes currently stored by the queue. | |
| `.queue.filled.pct` | Float (gauge) | How full the queue is relative to its maximum size, as a fraction from 0 to 1.   | Low throughput while `queue.filled.pct` is low means congestion in the input. Low throughput while `queue.filled.pct` is high means congestion in the output. |
| `.queue.added.events` | Integer | Number of events added to the queue by input workers. | |
| `.queue.added.bytes` | Integer | Number of bytes added to the queue by input workers. | |
| `.queue.consumed.events` | Integer | Number of events sent to output workers. | |
| `.queue.consumed.bytes` | Integer | Number of bytes sent to output workers. | |
| `.queue.removed.events` | Integer | Number of events removed from the queue after being processed by output workers. | |
| `.queue.removed.bytes` | Integer | Number of bytes removed from the queue after being processed by output workers. | |

When using the memory queue, byte metrics are only set if the output supports them. Currently only the Elasticsearch output supports byte metrics.

| Field path (relative to `.monitoring.metrics.filebeat`) | Type | Meaning | Troubleshooting hints |
| --- | --- | --- | --- |
| `.events.active` | Integer | Number of events being actively processed by {{filebeat}} (including events {{filebeat}} has already sent to the libbeat publisher pipeline, but not including events the pipeline has sent to the output). | If this number grows over time, it may indicate that {{filebeat}} inputs are harvesting events too fast for the pipeline and output to keep up. |


## Useful commands [_useful_commands]


### Parse monitoring metrics from unstructured Filebeat logs [_parse_monitoring_metrics_from_unstructured_filebeat_logs]

For Filebeat versions that emit unstructured logs, the following script can be used to parse monitoring metrics from such logs: [https://github.com/elastic/beats/blob/main/script/metrics_from_log_file.sh](https://github.com/elastic/beats/blob/main/script/metrics_from_log_file.sh).


### Check if {{filebeat}} is processing events [_check_if_filebeat_is_processing_events]

```
$ cat beat.log | jq -r '[.["@timestamp"],.monitoring.metrics.filebeat.events.active,.monitoring.metrics.libbeat.pipeline.events.active,.monitoring.metrics.libbeat.output.events.total,.monitoring.metrics.libbeat.output.events.acked,.monitoring.metrics.libbeat.output.events.failed//0] | @tsv' | sort
```

Example output:

The columns here are:

1. `.@timestamp`
2. `.monitoring.metrics.filebeat.events.active`
3. `.monitoring.metrics.libbeat.pipeline.events.active`
4. `.monitoring.metrics.libbeat.output.events.total`
5. `.monitoring.metrics.libbeat.output.events.acked`
6. `.monitoring.metrics.libbeat.output.events.failed`

```
2023-07-14T11:24:36.811Z	1	1	38033	38033	0
2023-07-14T11:25:06.811Z	1	1	17	17	0
2023-07-14T11:25:36.812Z	1	1	16	16	0
2023-07-14T11:26:06.811Z	1	1	17	17	0
2023-07-14T11:26:36.811Z	2	2	21	21	0
2023-07-14T11:27:06.812Z	1	1	18	18	0
2023-07-14T11:27:36.811Z	1	1	17	17	0
2023-07-14T11:28:06.811Z	1	1	18	18	0
2023-07-14T11:28:36.811Z	1	1	16	16	0
2023-07-14T11:37:06.811Z	1	1	270	270	0
2023-07-14T11:37:36.811Z	1	1	16	16	0
2023-07-14T11:38:06.811Z	1	1	17	17	0
2023-07-14T11:38:36.811Z	1	1	16	16	0
2023-07-14T11:41:36.811Z	3	3	323	323	0
2023-07-14T11:42:06.811Z	3	3	17	17	0
2023-07-14T11:42:36.812Z	4	4	18	18	0
2023-07-14T11:43:06.811Z	4	4	17	17	0
2023-07-14T11:43:36.811Z	2	2	17	17	0
2023-07-14T11:47:06.811Z	0	0	117	117	0
2023-07-14T11:47:36.811Z	2	2	14	14	0
2023-07-14T11:48:06.811Z	3	3	17	17	0
2023-07-14T11:48:36.811Z	2	2	17	17	0
2023-07-14T12:49:36.811Z	3	3	2008	1960	48
2023-07-14T12:50:06.812Z	2	2	18	18	0
2023-07-14T12:50:36.811Z	5	5	48	48	0
```
