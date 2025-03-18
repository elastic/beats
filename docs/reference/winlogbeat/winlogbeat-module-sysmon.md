---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/winlogbeat-module-sysmon.html
---

# Sysmon Module [winlogbeat-module-sysmon]

The sysmon module processes event log records from the [Sysinternals System Monitor](https://docs.microsoft.com/en-us/sysinternals/downloads/sysmon) (Sysmon) which is a Windows service and device driver that logs system activity to the event log. Sysmon is not bundled with Windows or Winlogbeat and must be installed independently.

The default configuration file includes configuration for the Sysmon channel. If you do not have Sysmon installed Winlogbeat will log a warning that it could not read from the `Microsoft-Windows-Sysmon/Operational` channel. It will continue to read from the other configured channels. If you do install Sysmon at a later time then a restart of Winlogbeat is required to make it begin reading from the channel.

This module was built based on Sysmon v13 event manifests. It contains transformations for each of the defined event IDs.


## Configuration [_configuration_4]

```yaml
winlogbeat.event_logs:
  - name: Microsoft-Windows-Sysmon/Operational

output.elasticsearch.pipeline: winlogbeat-%{[agent.version]}-routing <1>
```

1. All module processing is handled via Elasticsearch Ingest Node pipelines. See [Setup of Ingest Node pipelines](/reference/winlogbeat/winlogbeat-modules.md#winlogbeat-modules-setup) for details.


