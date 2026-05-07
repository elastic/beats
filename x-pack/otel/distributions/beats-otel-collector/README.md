# Beats OpenTelemetry Collector Distribution

**This distribution is used to ensure compatibility between Beats components and the OpenTelemetry collector builder. It is not intended for public release.**

## Components Included

### Receivers
- **filebeatreceiver** - Filebeat receiver
- **metricbeatreceiver** - Metricbeat receiver
- **filelogreceiver** - File log receiver
- **hostmetricsreceiver** - Host metrics receiver
- **httpcheckreceiver** - HTTP check receiver
- **jaegerreceiver** - Jaeger receiver
- **prometheusreceiver** - Prometheus receiver
- **otlp** - OTLP receiver
- **nop** - No-op receiver
- **zipkinreceiver** - Zipkin receiver

### Processors
- **beat** - Beats processor
- **attributes** - Attributes processor
- **batch** - Batch processor
- **cumulativetodelta** - Cumulative to delta processor
- **filter** - Filter processor
- **k8sattributes** - Kubernetes attributes processor
- **memory_limiter** - Memory limiter processor
- **resourcedetection** - Resource detection processor
- **resource** - Resource processor
- **transform** - Transform processor

### Exporters
- **elasticsearch** - Elasticsearch exporter
- **kafka** - Kafka exporter
- **debug** - Debug exporter
- **file** - File exporter
- **nop** - No-op exporter
- **otlp** - OTLP gRPC exporter
- **otlphttp** - OTLP HTTP exporter

### Extensions
- **beatsauth** - Beats authentication extension
- **basicauth** - Basic auth extension
- **bearertokenauth** - Bearer token auth extension
- **file_storage** - File storage extension
- **health_check** - Health check extension
- **memory_limiter** - Memory limiter extension
- **pprof** - pprof extension

### Config Providers
- **env** - Environment variable provider
- **file** - File provider
- **http** - HTTP provider
- **https** - HTTPS provider
- **yaml** - YAML provider

## Contributing

All Beats components should be added to this distribution's manifest to ensure integration with OpenTelemetry Collector binaries. 

To add a new Beats component located at `./x-pack/otel/extension/customextension`, append the following entries to the [manifest.yaml](./manifest.yaml) file:

```yaml
receivers:
  - gomod: github.com/elastic/beats/v7/x-pack/otel/extension/customextension v0.0.0

# Add to replaces section:
replaces:
  - github.com/elastic/beats/v7/x-pack/otel/extension/customextension => ../extension/customextension
```

Then add the component to the example [configuration file](./config.yaml).

## Building

From the `x-pack/otel` directory, use the mage target:

```bash
mage buildOtelDistro
```

This requires [ocb](https://opentelemetry.io/docs/collector/extend/ocb/) to be installed and available on `PATH`.

## Running

Run the collector with the example [config.yaml](./config.yaml):

```bash
mage runOtelDistro
```

To use a custom config file, pass it via `OTEL_ARGS`:

```bash
OTEL_ARGS="--config /path/to/config.yaml" mage runOtelDistro
```