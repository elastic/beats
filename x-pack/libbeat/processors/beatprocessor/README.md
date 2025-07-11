# Beat Processor

| Status    |                     |
| --------- | ------------------- |
| Stability | [development]: logs |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development

The Beat processor (`beat`) makes the functionality of [Beat processors] available in an OpenTelemetry Collector's processor.
This allows users to configure Beat processors anywhere in the OpenTelemetry Collector's pipeline, independently of Beat receivers.

> [!NOTE]
> This component is currently in development and no functionality is implemented.
> Including it in a pipeline is a no-op.
> The documentation describes the intended state after the functionality is implemented.

## Example

The following [Filebeat receiver] configuration

```yaml
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: host-logs
          paths:
            - /var/log/*.log
    processors:
      - add_host_metadata: ~
    output:
      otelconsumer:
```

is functionally equivalent to this one, using the Beat processor:

```yaml
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: host-logs
          paths:
            - /var/log/*.log
    output:
      otelconsumer:

processors:
  beat:
    processors:
      - add_host_metadata: ~
```

[Beat processors]: https://www.elastic.co/docs/reference/beats/filebeat/filtering-enhancing-data#using-processors
[Filebeat receiver]: https://github.com/elastic/beats/tree/main/x-pack/filebeat/fbreceiver
