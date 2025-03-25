---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/winlogbeat-module-security.html
---

# Security Module [winlogbeat-module-security]

The security module processes event log records from the Security log.


## Configuration [_configuration_3]

```yaml
winlogbeat.event_logs:
  - name: Security

output.elasticsearch.pipeline: winlogbeat-%{[agent.version]}-routing <1>
```

1. All module processing is handled via Elasticsearch Ingest Node pipelines. See [Setup of Ingest Node pipelines](/reference/winlogbeat/winlogbeat-modules.md#winlogbeat-modules-setup) for details.


