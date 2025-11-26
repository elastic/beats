# Opentelemetry metrics

## Summary

Open Telemetry metrics are collected at both the periodic run level and per
CEL program execution. OTEL metrics are exported at the end of each periodic run
(at the end of each interval). Metrics are collected as 'delta' metrics. 'Delta'
metrics are reset after each export. Each exported metric set contains the data
for a single periodic run.

Each CEL input instance has its own set of OTEL resource attributes. These are
key/value pairs that uniquely identify the CEL input instance.

## Enabling OTEL metrics collection and export
The export of OTEL metrics are off by default. OTEL metrics are enabled through
environment variables.

OTEL metrics can be sent to an otlp endpoint or to the console for testing.
CEL OTEL metrics are designed to be sent to an
[Elastic Cloud Mangaged OTLP Endpoint](https://www.elastic.co/docs/reference/opentelemetry/motlp). The histograms are sent as
Exponential Histograms. Metrics should be able to be sent to any OTLP endpoint
that can consume Exponential Histograms.

The default OTLP metrics protocol when enabled with an
OTEL_EXPORTER_OTLP_ENDPOINT is 'grpc'. 'http/protobuf' is also supported.
'http/json' is not supported.

To export OTEL metrics to a grpc OTLP endpoint set these environment variables.
``` text
Required:
export OTEL_EXPORTER_OTLP_ENDPOINT=<value>

Required if the OTLP is authenticated
export OTEL_EXPORTER_OTLP_HEADERS=<value>

Optional but suggested. OTEL_RESOURCE_ATTRIBUTES are added to the CEL input
instance resource attributes.
export OTEL_RESOURCE_ATTRIBUTES=service.name=<app-name>,service.version=<app-version>,deployment.environment=<env>
```

To use the httt/protobuf protocol include this environment variable with the
previously described set.
``` text
OTEL_EXPORTER_OTLP_METRICS_PROTOCOL="httt/protobuf"
```

To export OTEL metrics to the console set these environment variables.
``` text
export OTEL_METRICS_EXPORTER=console
```
## Exported metrics

Each CEL input has an associated Open Telemetry Resource associate with it.
Resource attributes that are included for every CEL input instance
semconv.ServiceInstanceID(env.IDWithoutName),
attribute.String("package.name", cfg.GetPackageStringValue("name")),
attribute.String("package.version", cfg.GetPackageStringValue("version")),
attribute.String("package.datastream", cfg.DataStream),
attribute.String("agent.version", env.Agent.Version),
attribute.String("agent.id", env.Agent.ID.String())}
| name                                | description                         |
|-------------------------------------|-------------------------------------|
| agent.version                       | version of agent                    |
| agent.id              | agent id              |
| service.instance.id                 | id of the cel input instance |
| package.name | name of the package|
| package.version | version of the package|
| package.datastream | the datastream name in the package|

Resource attributes that may be passed in an OTEL_RESOURCE_ATTRIBUTES
environment variable and added to the CEL input instance resource attributesexport
OTEL_EXPORTER_OTLP_METRICS_PROTOCOL="httt/protobuf"
export OTEL_METRICS_EXPORTER=console

## Exported Metrics

### Resource attributes
Each cel input has an associated Open Telemetry resource associated with it.
These resource attributes that are included for every cel input instance.

| name                                | description                         |
|-------------------------------------|-------------------------------------|
| agent.version                       | version of agent                    |
| agent.id              | agent id              |
| service.instance.id                 | id of the cel input instance |
| package.name | name of the package|
| package.version | version of the package|
| package.datastream | the datastream name in the package|

Resource attributes that are defined in an OTEL_RESOURCE_ATTRIBUTES
environment variable will be added to the CEL input instance.
These attributes are expected.

|  name  |  description   | example   |
|---|---|---|
| service.name  | service that is running the program  | elastic-agent  |
| deployment.environment | deployment environment  | production


### Metrics

These metrics are generated in the cel_metrics.go file and are scoped in the OTEL metrics as 'github.com/elastic/beats/x-pack/filebeat/otel/cel_metrics.go'

| name                                      | description                                                                            | metric type      |
|-------------------------------------------|----------------------------------------------------------------------------------------|------------------|
| input.cel.periodic.run                    | the number of times a periodic run was started.                                        | Int64Counter     |
| input.cel.periodic.program.run.started    | the number of times a program was started in a periodic run.                           | Int64counter     |
| input.cel.periodic.program.run.success    | the number of times a program terminated without an error in a periodic run.           | Int64counter     |
| input.cel.periodic.batch.generated        | the number of the number of batches generated in a periodic run.                       | Int64counter     |
| input.cel.periodic.batch.published        | the number of the number of batches successfully published in a periodic run.          | Int64counter     |
| input.cel.periodic.event.generated        | the number of the number of events generated in a periodic run.                        | Int64counter     |
| input.cel.periodic.event.published        | the number of the number of events published in a periodic run.                        | Int64counter     |
| input.cel.periodic.run.duration           | the total duration of time in seconds spent in a periodic run.                         | Float64Counter   |
| input.cel.periodic.cel.duration           | the total duration of time in seconds spent processing CEL programs in a periodic run. | Float64Histogram |
| input.cel.periodic.event.publish.duration | the total duration of time in seconds publishing events in a periodic run.             | Float64Histogram |
| input.cel.program.batch                   | the number of batches the program has generated.                                       | Int64Histogram   |
| input.cel.program.event                   | the number of events the program has generated.                                        | Int64Histogram   |
| input.cel.program.event.published         | the number of events the program has published.                                        | Int64Histogram   |
| input.cel.program.batch.published         | the number of batched the program has published.                                       | Int64Histogram   |
| input.cel.program.run.duration            | the total time in seconds spent executing the program.                                 | Float64Histogram |
| input.cel.program.cel.duration            | the total time in seconds spent processing the CEL program.                            | Float64Histogram |
| input.cel.program.publish.duration        | the total time in seconds spent publishing in the program.                             | Float64Histogram |

These metrics are generated by the OTEL SDK through wrapping the transport and are scoped in the OTEL metrics as
'go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp'.
A metric is exported for each unique set of attributes:
- http.response.status_code
- network.protocol.name
- network.protocol.version
- server.address
- server.port
- url.scheme


| name                         | description                                  | metric type      |
|------------------------------|----------------------------------------------|------------------|
| http.client.request.duration | The duration in seconds for an http request  | Float64Histogram |
| http.client.request.body.size | The size of the request in bytes             | Int64Histogram   |


