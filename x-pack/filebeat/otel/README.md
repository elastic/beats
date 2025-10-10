# Opentelemetry metrics

The CEL input is currently able to export Open Telemetry metrics.
The export of OTEL metrics are off by default. The control of OTEL
exporting is through environment variables.

## Configuration
OTEL metrics can be sent to an otlp endpoint or to console for testing.
OTEL metrics endpoints [Open Telemetry Collector](https://www.elastic.co/docs/reference/opentelemetry), the
[Elastic Cloud Mangaged OTLP Endpoint](https://www.elastic.co/docs/reference/opentelemetry/motlp) in Elastic Cloud
or the [otlp endpoint in Elastic APM](https://www.elastic.co/docs/solutions/observability/apm/use-opentelemetry-with-apm).

To export OTEL metrics to console set these environment variables.
```
unset OTEL_EXPORTER_OTLP_ENDPOINT
unset OTEL_EXPORTER_OTLP_HEADERS
export OTEL_METRICS_EXPORTER=console
```
To export OTEL metrics to an OTLP endpoint set these environment variables,
```
export OTEL_EXPORTER_OTLP_ENDPOINT=<endpoint URL>
export OTEL_METRICS_EXPORTER=otlp

Authorization headers will depend upon the endpoint being used.
export OTEL_EXPORTER_OTLP_HEADERS="Authorization=ApiKey <key>"
```

The console exports in JSON. The default protocol for OTLP is GRPC.
Filebeat also supports "http/protobuf". It does not support "http/json."
To use an http/protobuf protocol:

```
export OTEL_EXPORTER_OTLP_METRICS_PROTOCOL="http/protobuf"
```
## Exported metrics

Each CEL input has an associated Open Telemetry Resource associate with it

|name|description|
|----|---|
|resource.attributes.service.name | the package name of the integration|
|resource.attributes.service.version | version of the integration|
|resource.attributes.agent.id  | id of the agent|
|resource.attributes.instance.id ||

Exported metrics:
A program run is single run of the cel program. A periodic run is all the
program runs for that periodic run.

|name|description|metric type|
|---|---|---|
|input.cel.periodic.run.count              | the number of times a periodic run was started.|Int64Counter|
| input.cel.periodic.program.started       | a histogram of times a program was started in a periodic run.|Int64Histogram|
| input.cel.periodic.program.success       | a histogram of times a program terminated without an error in a periodic run.|Int64Histogram|
| input.cel.periodic.batch.generated       | a histogram of the number of batches generated in a periodic run.|Int64Histogram|
| input.cel.periodic.batch.published       | a histogram of the number of batches successfully published in a periodic run.|Int64Histogram|
| input.cel.periodic.event.generated       | a histogram of the number of events generated in a periodic run.|Int64Histogram|
| input.cel.periodic.event.published       | a histogram of the number of events published in a periodic run.|Int64Histogram|
| input.cel.periodic.run.duration          | a histogram of the total duration of time in seconds spent in a periodic run.|Int64Histogram|
| input.cel.periodic.cel.duration          | a histogram of the total duration of time in seconds spent processing CEL programs in a periodic run.|Int64Histogram|
|input.cel.periodic.event.publish.duration | a histogram of the total duration of time in seconds publishing events in a periodic run.|Int64Histogram|
|input.cel.program.run.started.count      | a count of the number of times a program was started.|Int64Counter|
| input.cel.program.run.success.count      | a count of the number of times a program terminated without error.|Int64Counter|
| input.cel.program.batch.count            | a count of the number of batches the program has generated.|Int64Counter|
| input.cel.program.event.count            | a count of the number of events the program has generated.|Int64Counter|
| input.cel.program.event.published.count  | a count of the number of events the program has published.|Int64Counter|
| input.cel.program.batch.published.count  | a count of the number of batched the program has published.|Int64Counter|
| input.cel.program.run.duration           | a histogram of the total time in seconds spent executing the program.|Int64Histogram|
| input.cel.program.cel.duration           | a histogram of the time in seconds spent processing the CEL program.|Int64Histogram|
| input.cel.program.publish.duration       | a histogram of the time in seconds spent publishing in the program.|Int64Histogram|

