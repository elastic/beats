---
navigation_title: "rate_limit"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/rate-limit.html
---

# Rate limit the flow of events [rate-limit]


The `rate_limit` processor limits the throughput of events based on the specified configuration.

In the current implementation, rate-limited events are dropped. Future implementations may allow rate-limited events to be handled differently.

```yaml
processors:
- rate_limit:
   limit: "10000/m"
```

```yaml
processors:
- rate_limit:
   fields:
   - "cloudfoundry.org.name"
   limit: "400/s"
```

```yaml
processors:
- if.equals.cloudfoundry.org.name: "acme"
  then:
  - rate_limit:
      limit: "500/s"
```

The following settings are supported:

`limit`
:   The rate limit. Supported time units for the rate are `s` (per second), `m` (per minute), and `h` (per hour).

`fields`
:   (Optional) List of fields. The rate limit will be applied to each distinct value derived by combining the values of these fields.

