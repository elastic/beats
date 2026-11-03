# libbeat - Framework for building the Beats

libbeat is a Go library containing the common packages for all the
[Beats](https://www.elastic.co/beats).
It is Apache licensed and actively maintained by the Elastic team.

If you want to create a new project that reads some sort of operational data
and ships it to Elasticsearch, we suggest you make use of this library. Please
start by reading our [CONTRIBUTING](../CONTRIBUTING.md) file. We also have a
[developer
guide](https://www.elastic.co/guide/en/beats/devguide/master/index.html) to
help you with the creation of new Beats.

Please also open a topic on the [forums](https://discuss.elastic.co/c/beats/libbeat) and
we'll help you get started.

## Building without the OpenTelemetry output

Referencing `libbeat/publisher/pipeline` normally pulls in
`libbeat/otel/otelconsumer` and its OpenTelemetry dependencies, because the
pipeline's OTel output controller imports it. Downstream projects that embed
libbeat but never run a Beat as an OpenTelemetry receiver can drop that part of
the module graph with the `nooteloutput` build tag:

```shell
go build -tags nooteloutput ./...
```

With the tag set, `libbeat/publisher/pipeline/output_otel.go` is excluded and a
stub replaces the OTel output controller, which takes `otelconsumer` and most
of its collector dependencies out of the package's transitive imports. It does
not remove OpenTelemetry entirely: `libbeat/beat` itself depends on
`collector/pdata` and `collector/consumer`, so those remain either way. That
residual coupling is why the tag is described as a temporary workaround in the
commit that added it, rather than a long-term API, so treat it as subject to
change.

Only the receiver pipeline (`pipeline.NewForReceiver`) uses the OTel output
controller, so a Beat that publishes through a normal output is unaffected.
Do not set the tag if you use `pipeline.NewForReceiver`: the stub returns no
controller, leaving the pipeline without an output.

To see the Beats that are built on top of this library, please see the list
from the main [README](../README.md) file and the list of [community
Beats](https://www.elastic.co/guide/en/beats/libbeat/master/community-beats.html).
