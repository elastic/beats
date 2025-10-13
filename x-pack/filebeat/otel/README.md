# Opentelemetry metrics

The CEL input is currently able to export Open Telemetry metrics.
The export of OTEL metrics are off by default. The control of OTEL
exporting is through environment variables.

## Configuration
OTEL metrics can be sent to an otlp endpoint or to console for testing.
OTEL metrics endpoints [Open Telemetry Collector](https://www.elastic.co/docs/reference/opentelemetry), the
[Elastic Cloud Mangaged OTLP Endpoint](https://www.elastic.co/docs/reference/opentelemetry/motlp) in Elastic Cloud
or the [otlp endpoint in Elastic APM](https://www.elastic.co/docs/solutions/observability/apm/use-opentelemetry-with-apm).

To export OTEL metrics to an OTLP endpoint set these environment variables.
In production the APM server UI will display the value that should be used
```
export OTEL_EXPORTER_OTLP_ENDPOINT=<value>
export OTEL_RESOURCE_ATTRIBUTES=service.name=<app-name>,service.version=<app-version>,deployment.environment=production
export OTEL_EXPORTER_OTLP_HEADERS=<value>
```

To export OTEL metrics to console set these environment variables.
```
unset OTEL_EXPORTER_OTLP_ENDPOINT
unset OTEL_EXPORTER_OTLP_HEADERS
unset OTEL_RESOURCE_ATTRIBUTES
export OTEL_METRICS_EXPORTER=console
```

The console exports in JSON. The default protocol for OTLP is gRPC.
Filebeat also supports "http/protobuf". It does not support "http/json."
To use an http/protobuf protocol:

```
export OTEL_EXPORTER_OTLP_METRICS_PROTOCOL="http/protobuf"
```
## Exported metrics

Each CEL input has an associated Open Telemetry Resource associate with it

| name                                | description                         |
|-------------------------------------|-------------------------------------|
| agent.version                       | version of agent                    |
| deployment.environment              | deployment environment              |
| service.instance.id                 | id of the input                     |
| service.name                        | package name of the integration     |
| service.version                     | package version of the integration  |


Exported metrics:
A program run is single run of the cel program. A periodic run is all the
program runs for that periodic run.

| name                                       |description|metric type|
|--------------------------------------------|---|---|
| input.cel.periodic.run.count               | the number of times a periodic run was started.|Int64Counter|
| input.cel.periodic.program.started         | a count of the number of times a program was started in a periodic run.|Int64count of the number|
| input.cel.periodic.program.success         | a count of the number of times a program terminated without an error in a periodic run.|Int64count of the number|
| input.cel.periodic.batch.generated         | a count of the number of the number of batches generated in a periodic run.|Int64count of the number|
| input.cel.periodic.batch.published         | a count of the number of the number of batches successfully published in a periodic run.|Int64count of the number|
| input.cel.periodic.event.generated         | a count of the number of the number of events generated in a periodic run.|Int64count of the number|
| input.cel.periodic.event.published         | a count of the number of the number of events published in a periodic run.|Int64count of the number|
| input.cel.periodic.run.duration            | a count of the number of the total duration of time in seconds spent in a periodic run.|Int64count of the number|
| input.cel.periodic.cel.duration            | a count of the number of the total duration of time in seconds spent processing CEL programs in a periodic run.|Int64count of the number|
| input.cel.periodic.event.publish.duration  | a count of the number of the total duration of time in seconds publishing events in a periodic run.|Int64count of the number|
| input.cel.program.run.started.count        | a count of the number of times a program was started.|Int64Counter|
| input.cel.program.run.success.count        | a count of the number of times a program terminated without error.|Int64Counter|
| input.cel.program.batch.count              | a count of the number of batches the program has generated.|Int64Counter|
| input.cel.program.event.count              | a count of the number of events the program has generated.|Int64Counter|
| input.cel.program.event.published.count    | a count of the number of events the program has published.|Int64Counter|
| input.cel.program.batch.published.count    | a count of the number of batched the program has published.|Int64Counter|
| input.cel.program.run.duration             | a count of the number of the total time in seconds spent executing the program.|Int64count of the number|
| input.cel.program.cel.duration             | a count of the number of the time in seconds spent processing the CEL program.|Int64count of the number|
| input.cel.program.publish.duration         | a count of the number of the time in seconds spent publishing in the program.|Int64count of the number|

