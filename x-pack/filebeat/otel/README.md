# Opentelemetry metrics

The CEL input is currently able to export Open Telemetry metrics. The OTEL
metrics are collected at the total periodic run level as well as per
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

If OTEL_RESOURCE_ATTRIBUTES
is in the environment, the key value pairs will be added to the input's
Resource Attributes ans sent with the metrics.

```

To export OTEL metrics to console set these environment variables.
```
unset OTEL_EXPORTER_OTLP_ENDPOINT
unset OTEL_EXPORTER_OTLP_HEADERS
export OTEL_METRICS_EXPORTER=console

setting OTEL_RESOURCE_ATTRIBUTES is optional
```

The console exports in JSON. The default protocol for OTLP is gRPC.
Filebeat also supports "http/protobuf". It does not support "http/json"
because the Go SDK does not support it. The http protocol is included
for endpoints other than the elastic OTEL OTLP endpoint

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

| name                                       |description| metric type    |
|--------------------------------------------|---|----------------|
| input.cel.periodic.run.count               | the number of times a periodic run was started.| Int64Counter   |
| input.cel.periodic.program.started         | the number of times a program was started in a periodic run.| Int64counter   |
| input.cel.periodic.program.success         | the number of times a program terminated without an error in a periodic run.| Int64counter   |
| input.cel.periodic.batch.generated         | the number of the number of batches generated in a periodic run.| Int64counter   |
| input.cel.periodic.batch.published         | the number of the number of batches successfully published in a periodic run.| Int64counter   |
| input.cel.periodic.event.generated         | the number of the number of events generated in a periodic run.| Int64counter   |
| input.cel.periodic.event.published         | the number of the number of events published in a periodic run.| Int64counter   |
| input.cel.periodic.run.duration            | the total duration of time in seconds spent in a periodic run.| Float64Counter   |
| input.cel.periodic.cel.duration            | the total duration of time in seconds spent processing CEL programs in a periodic run.| Float64Counter   |
| input.cel.periodic.event.publish.duration  | the total duration of time in seconds publishing events in a periodic run.| Float64Counter   |
| input.cel.program.run.started.count        | the number of times a program was started.| Int64Counter   |
| input.cel.program.run.success.count        | the number of times a program terminated without error.| Int64Counter   |
| input.cel.program.batch.count              | the number of batches the program has generated.| Int64Counter   |
| input.cel.program.event.count              | the number of events the program has generated.| Int64Counter   |
| input.cel.program.event.published.count    | the number of events the program has published.| Int64Counter   |
| input.cel.program.batch.published.count    | the number of batched the program has published.| Int64Counter   |
| input.cel.program.run.duration             | the total time in seconds spent executing the program.| Float64Counter   |
| input.cel.program.cel.duration             | the total time in seconds spent processing the CEL program.| Float64Counter   |
| input.cel.program.publish.duration         | the total time in seconds spent publishing in the program.| Float64Counter |

