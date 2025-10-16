# OpenTelemetry Collector Components in Beats

This is the home of OpenTelemetry Collector components like receivers, processors, exporters, extensions, connectors, etc. that are related to Beats.

The intended structure of this directory is to mimic the structure in the [OpenTelemetry Collector Contrib] repository, specifically:

- Put X receiver in `receiver/xreceiver` subdirectory,
- Put Y processor in `processor/yprocessor` subdirectory,
- Put Z exporter in `exporter/zexporter` subdirectory,
- and so on.

There should be no need to put any Go files directly in this directory.
