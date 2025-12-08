# Logstash Exporter

| Status    |                     |
|-----------|---------------------|
| Stability | [development]: logs |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development

> [!NOTE]
> This component is currently in development and no functionality is implemented.
> Including it in a pipeline is a no-op.
> The documentation describes the intended state after the functionality is implemented.

The Logstash exporter (`logstash`) is an OpenTelemetry Collector exporter that wraps the Beats [Logstash output] and allows
you to send data to Logstash by using the lumberjack protocol, which runs over TCP.

> [!NOTE]
> This component is highly coupled with the Beats ecosystem, and is not designed to be used with
> the native OpenTelemetry Collector.

## Configuration options


[Logstash output]: https://www.elastic.co/docs/reference/beats/filebeat/logstash-output
