---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/error-event-structure.html
---

# Error event structure [error-event-structure]

Metricbeat sends an error event when the service is not reachable. The error event has the same structure as the [base event](/reference/metricbeat/metricbeat-event-structure.md), but also has an error field that contains an error string. This makes it possible to check for errors across all metric events.

The following example shows an error event sent when the Apache server is not reachable:

```json
{
  "@timestamp": "2016-03-18T12:18:57.124Z",
  "apache-status": {},
  "beat": {
    "hostname": "host.example.com",
    "name": "host.example.com"
  },
  "error": {
    "message": "Get http://127.0.0.1/server-status?auto: dial tcp 127.0.0.1:80: getsockopt: connection refused",
  },
  "metricset": {
    "module": "apache",
    "name": "status",
    "rtt": 1082
  },
  .
  .
  .

  "type": "metricsets"
```

