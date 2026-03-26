# Beats OpenTelemetry Collector Distribution

This distribution contains OpenTelemetry Collector components integrated with Elastic Beats components.

**This distribution is used to ensure compatibility between Beats components and the OpenTelemetry collector builder. It is not intended for public release.**

## Components Included

### Receivers
- **filebeatreceiver** - Filebeat otel receiver
- **metricbeatreceiver** - Metricbeat otel receiver
- Standard OpenTelemetry receivers (OTLP, Jaeger, Prometheus, etc.)

### Processors
- **beat** - Beats processors
- Standard OpenTelemetry processors (batch, attributes, resource, etc.)

### Exporters
- **logstash** - Export data to Logstash
- **elasticsearchexporter** - Export data to Elasticsearch
- Standard OpenTelemetry exporters (OTLP, debug, etc.)

### Extensions
- **beatsauth** - Beats authentication extension
- **elasticsearch_storage** - Elasticsearch storage extension
- Standard OpenTelemetry extensions (health check, pprof, etc.)

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

This distribution can be built using the [OpenTelemetry Collector Builder (OCB)](https://opentelemetry.io/docs/collector/extend/ocb/):

```bash
ocb --config manifest.yaml
```

## Testing

The components can be tested using the configuration in [config.yaml](./config.yaml).

```
cd _build
./beats-otel-collector --config ../config.yaml
```