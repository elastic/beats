# Beat Processor

| Status    |                     |
| --------- | ------------------- |
| Stability | [development]: logs |

[development]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#development

The Beat processor (`beat`) is an OpenTelemetry Collector processor that wraps the [Beat processors].
This allows you to use Beat processors like e.g. [add_host_metadata] anywhere in the OpenTelemetry Collector's pipeline, independently of Beat receivers.

> [!NOTE]
> This component is only expected to work correctly with data from the Beat receivers: [Filebeat receiver], [Metricbeat receiver].
> This is because it relies on the specific structure of telemetry emitted by those components.
> Using it with data coming from other components is not recommended and may result in unexpected behavior.

Here are the currently supported processors:

- [add_cloud_metadata]
- [add_docker_metadata]
- [add_fields]
- [add_host_metadata]
- [add_kubernetes_metadata]

## Default processors in Beat receivers

The Beat receivers have a set of default processors that are included when the `processors` option is not specified.
These processors are: [add_cloud_metadata], [add_docker_metadata], [add_host_metadata], [add_kubernetes_metadata].
To disable them, explicitly specify the `processors` configuration option of the Beat receiver.
The list of processors can be an empty list or an arbitrary list of processors.

For example:

```yaml
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: host-logs
          paths:
            - /var/log/*.log
```

The above Filebeat receiver configuration does not explicitly specify the `processors` option.
In this case, the four processors listed above are included and ran as part of the Filebeat receiver.

```yaml
receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: host-logs
          paths:
            - /var/log/*.log
    processors: []
```

The above Filebeat receiver configuration specifies an empty list of processors.
In this case, none of the default processors are ran as part of the Filebeat receiver.

## Examples

The following OpenTelemetry Collector configuration using only the [Filebeat receiver]:

```yaml
service:
  pipelines:
    logs:
      receivers: [filebeatreceiver]
      exporters: [debug]

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
<<<<<<< HEAD

exporters:
  debug:
=======
>>>>>>> b80d1e914 ([beatreceiver] Do not require specifiying otelconsumer output (#47693))
```

is functionally equivalent to this one, using the Beat processor:

```yaml
service:
  pipelines:
    logs:
      receivers: [filebeatreceiver]
      processors: [beat]
      exporters: [debug]

receivers:
  filebeatreceiver:
    filebeat:
      inputs:
        - type: filestream
          id: host-logs
          paths:
            - /var/log/*.log
    processors: []

processors:
  beat:
    processors:
      - add_host_metadata:
          netinfo:
            enabled: false

exporters:
  debug:
```

## Using the `add_cloud_metadata` processor

To use the [add_cloud_metadata] processor, configure the processor as follows:

```yaml
processors:
  beat:
    processors:
      - add_cloud_metadata:
```

You can configure the cloud metadata enrichment using the options supported by the [add_cloud_metadata] processor.

## Using the `add_docker_metadata` processor

To use the [add_docker_metadata] processor, configure the processor as follows:

```yaml
processors:
  beat:
    processors:
      - add_docker_metadata:
```

You can configure the Docker metadata enrichment using the options supported by the [add_docker_metadata] processor.

## Using the `add_fields` processor

To use the [add_fields] processor, configure the processor as follows:

```yaml
processors:
  beat:
    processors:
      - add_fields:
          fields:
            custom_field: custom-value
```

You can configure the processor using the options supported by the [add_fields] processor.

## Using the `add_host_metadata` processor

To use the [add_host_metadata] processor, configure the processor as follows:

```yaml
processors:
  beat:
    processors:
      - add_host_metadata:
```

You can configure the host metadata enrichment using the options supported by the [add_host_metadata] processor.

## Using the `add_kubernetes_metadata` processor

To use the [add_kubernetes_metadata] processor, configure the processor as follows:

```yaml
processors:
  beat:
    processors:
      - add_kubernetes_metadata:
          indexers:
            - container:
          matchers:
            - logs_path:
```

You can configure the Kubernetes metadata enrichment using the options supported by the [add_kubernetes_metadata] processor.

Note that you need to explicitly configure at least one [indexer][indexers] and at least one [matcher][matchers] for the enrichment to work.
In the example above, the `container` indexer and the `logs_path` matcher are configured.

[Beat processors]: https://www.elastic.co/docs/reference/beats/filebeat/filtering-enhancing-data#using-processors
[Filebeat receiver]: https://github.com/elastic/beats/tree/main/x-pack/filebeat/fbreceiver
[Metricbeat receiver]: https://github.com/elastic/beats/tree/main/x-pack/metricbeat/mbreceiver
[add_cloud_metadata]: https://www.elastic.co/docs/reference/beats/filebeat/add-cloud-metadata
[add_docker_metadata]: https://www.elastic.co/docs/reference/beats/filebeat/add-docker-metadata
[add_fields]: https://www.elastic.co/docs/reference/beats/filebeat/add-fields
[add_host_metadata]: https://www.elastic.co/docs/reference/beats/filebeat/add-host-metadata
[add_kubernetes_metadata]: https://www.elastic.co/docs/reference/beats/filebeat/add-kubernetes-metadata
[indexers]: https://www.elastic.co/docs/reference/beats/filebeat/add-kubernetes-metadata#_indexers
[matchers]: https://www.elastic.co/docs/reference/beats/filebeat/add-kubernetes-metadata#_matchers
