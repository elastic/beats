# Logstash Exporter

| Status    |                     |
| --------- | ------------------- |
| Stability | [development]: logs |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development

The Logstash exporter (`logstash`) is an OpenTelemetry Collector exporter that wraps the Beats [Logstash output] and sends
log data to Logstash using the lumberjack protocol over TCP.

> [!NOTE]
> This component is only expected to work correctly with data from the Beat receivers: [Filebeat receiver], [Metricbeat receiver].
> Using it with data coming from other components is not recommended and may result in unexpected behavior.

## Configuration options

The exporter accepts the same configuration options as the Beats [Logstash output]. At minimum, `hosts` is required.

See the [Logstash output] documentation for the full list of options and their defaults.

## Example

```yaml
service:
  pipelines:
    logs:
      receivers: [filebeatreceiver]
      exporters: [logstash]

receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: host-logs
          paths:
            - /var/log/*.log

exporters:
  logstash:
    hosts: ["localhost:5044"]
```

[Logstash output]: https://www.elastic.co/docs/reference/beats/filebeat/logstash-output
[Filebeat receiver]: ../../../filebeat/fbreceiver
[Metricbeat receiver]: ../../../metricbeat/mbreceiver
