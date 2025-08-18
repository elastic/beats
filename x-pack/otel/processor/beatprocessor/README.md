# Beat Processor

| Status    |                     |
| --------- | ------------------- |
| Stability | [development]: logs |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development

> [!NOTE]
> This component is currently in development and functionality is limited.

The Beat processor (`beat`) is an OpenTelemetry Collector processor that wraps the [Beat processors].
This allows you to use Beat processors like e.g. [add_host_metadata] anywhere in the OpenTelemetry Collector's pipeline, independently of Beat receivers.

> [!NOTE]
> This component is only expected to work correctly with data from the Beat receivers: [Filebeat receiver], [Metricbeat receiver].
> This is because it relies on the specific structure of telemetry emitted by those components.
> Using it with data coming from other components is not recommended and may result in unexpected behavior.

The processor enriches the telemetry with host metadata by using the [add_host_metadata] processor under the hood.
Note that configuration is limited at this stage.
Host metadata is added unconditionally and cannot be disabled.
You can configure the host metadata enrichment using the options that the [add_host_metadata] processor allows.
The only exception is that the option `replace_fields` is always set to `true` and setting it to `false` has no effect.

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
      - add_host_metadata:
          netinfo:
            enabled: false
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
      - add_host_metadata:
          netinfo:
            enabled: false
```

[Beat processors]: https://www.elastic.co/docs/reference/beats/filebeat/filtering-enhancing-data#using-processors
[Filebeat receiver]: https://github.com/elastic/beats/tree/main/x-pack/filebeat/fbreceiver
[Metricbeat receiver]: https://github.com/elastic/beats/tree/main/x-pack/metricbeat/mbreceiver
[add_host_metadata]: https://www.elastic.co/docs/reference/beats/filebeat/add-host-metadata
