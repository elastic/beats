---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/metrics-winlogbeat.html
---

# Event Processing Metrics [metrics-winlogbeat]

Winlogbeat exposes metrics under the [HTTP monitoring endpoint](/reference/winlogbeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the event log processing activity of Winlogbeat.


## Winlog Metrics [_winlog_metrics]

| Metric | Description |
| --- | --- |
| `provider` | Name of the provider being read. |
| `received_events_total` | Total number of events received. |
| `discarded_events_total` | Total number of discarded events. |
| `errors_total` | Total number of errors. |
| `received_events_count` | Histogram of the number of events in each non-zero batch. |
| `source_lag_time` | Histogram of the difference in nanoseconds between timestamped event’s creation and reading. |
| `batch_read_period` | Histogram of the elapsed time in nanoseconds between non-zero batch reads. |

