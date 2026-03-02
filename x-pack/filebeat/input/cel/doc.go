// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

/*
Package cel implements an input that uses the Common Expression Language to
perform requests and do endpoint processing of events. The cel package exposes
the github.com/elastic/mito/lib CEL extension library.

# OpenTelemetry Metrics

The CEL input exports OpenTelemetry metrics at the end of each periodic run.
Each export captures metrics for that interval only; counters reset between exports.

Metrics export is disabled by default. Enable export to a OTLP/gRPC endpoint by setting environment variables:

  - OTEL_EXPORTER_OTLP_ENDPOINT: Required. The OTLP endpoint URL.
  - OTEL_EXPORTER_OTLP_HEADERS: Required if endpoint is authenticated.
  - OTEL_RESOURCE_ATTRIBUTES: Optional but recommended
  - OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION: Optional. Set to "explicit_bucket_histogram" to use
    explicit bucket histograms instead of the default exponential histograms. This is required for backends that
    do not support exponential histograms (e.g. Elastic APM Server).

See [otel.ExportFactory] for environment settings to run console or http/protobuf output.

Resource attributes for Open Telemetry CEL Input Metrics
Each CEL input has an associated Open Telemetry Resource associated with it. The Resource Attribute Set identifies a
unique metric set in Elastic. Changing the Resource Attribute Set for input will identify the input's metrics as
different metric set.

These Resource Attributes are included for every CEL input instance:

	Name                 Description

	agent.version        version of agent
	agent.id             the id of the agent
	service.instance.id  id of the cel input instance
	package.name         name of the integration package
	package.version      version of the integration package
	package.data_stream  the datastream name in the integration package

Resource Attributes that are defined in an OTEL_RESOURCE_ATTRIBUTES environment variable will be added to the CEL
input instance Resource Attributes set.  An example of an OTEL_RESOURCE_ATTRIBUTES:

	service.name=elastic-agent,service.version=9.1.2,deployment.environment=production

These attributes are expected in the OTEL_RESOURCE_ATTRIBUTES but not required:

	Name                     Description

	service.name            service that is running the program, ex. elastic-agent
	deployment.environment  deployment environment, ex. production

# Open Telemetry Metrics for CEL Input

CEL Metrics are sent as Delta metrics for each Periodic Run (Interval).
Export occurs as a push to an OTLP endpoint at the end of the Periodic Run. Metrics are reset between Periodic Runs.
Metrics are collected at several points during the CEL input periodic run see the diagram
https://github.com/elastic/beats/tree/main/x-pack/filebeat/input/cel/cel_metric_collection.png for collection points.

CEL Metrics exported for each periodic run:

		Name                                        Description                                                                              Metric Type

		input.cel.periodic.run                      the number of times a periodic run was started.                                          Int64Counter
		input.cel.periodic.program.run.started      the number of times a program was started in a periodic run.                             Int64Counter
		input.cel.periodic.program.run.success      the number of times a program terminated without an error in a periodic run.             Int64Counter
		input.cel.periodic.batch.received           the number of the number of batches generated in a periodic run.                         Int64Counter
		input.cel.periodic.batch.published          the number of the number of batches successfully published in a periodic run.            Int64Counter
		input.cel.periodic.event.received           the number of the number of events generated in a periodic run.                          Int64Counter
		input.cel.periodic.event.published          the number of the number of events published in a periodic run.                          Int64Counter
		input.cel.periodic.run.duration             the total duration of time in seconds spent in a periodic run.                           Float64Counter
		input.cel.periodic.cel.duration             the total duration of time in seconds spent processing CEL programs in a periodic run.   Float64Histogram
		input.cel.periodic.event.publish.duration   the total duration of time in seconds publishing events in a periodic run.               Float64Histogram
		input.cel.program.batch.received            the number of batches the program has generated.                                         Int64Histogram
	    input.cel.program.event.received            the number of events the program has generated.                                          Int64Histogram
	    input.cel.program.batch.published           the number of batches the program has published.                                         Int64Histogram
		input.cel.program.event.published           the number of events the program has published.                                          Int64Histogram
		input.cel.program.run.duration              the time in seconds spent executing the program.                                         Float64Histogram
		input.cel.program.cel.duration              the time in seconds spent processing the CEL program.                                    Float64Histogram
		input.cel.program.publish.duration          the time in seconds spent publishing in the program.                                     Float64Histogram

HTTP metrics are generated by the OTEL SDK through wrapping the transport and are scoped in the OTEL metrics as
' go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp'.
A metric is exported for each unique set of attributes:
  - http.request.method
  - http.response.status_code
  - network.protocol.name
  - network.protocol.version
  - server.address
  - server.port
  - url.scheme

HTTP Metrics exported by the Open Telemetry SDK:

	Name                            Description                                    Metric Type

	http.client.request.duration    The duration in seconds for an http request    Float64Histogram
	http.client.request.body.size   The size of the request in bytes               Int64Histogram

See cel_metric_collection.png for a diagram of when each OTEL CEL metric is collected.

# Input Metrics

Input Metrics are part of the agent monitoring framework. They are cumulative metrics for each input for the entire
run of a CEL input. They are packaged together with other agent data for monitoring.
Durations are collected as histograms and are in nanoseconds.
Cumulative HTTP metrics are collected through a wrapper on the HTTP transport.
See https://www.elastic.co/guide/en/beats/filebeat/current/http-endpoint.html
and https://www.elastic.co/docs/reference/fleet/monitor-elastic-agent for details
on agent monitoring.
*/
package cel
