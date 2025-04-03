---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-graphite.html
---

# Graphite module [metricbeat-module-graphite]

This is the Graphite module.

The default metricset is `server`.


## Example configuration [_example_configuration_28]

The Graphite module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: graphite
  metricsets: ["server"]
  enabled: true

  # Host address to listen on. Default localhost.
  #host: localhost

  # Listening port. Default 2003.
  #port: 2003

  # Protocol to listen on. This can be udp or tcp. Default udp.
  #protocol: "udp"

  # Receive buffer size in bytes
  #receive_buffer_size: 1024

  #templates:
  #  - filter: "test.*.bash.*" # This would match metrics like test.localhost.bash.stats
  #    namespace: "test"
  #    template: ".host.shell.metric*" # test.localhost.bash.stats would become metric=stats and tags host=localhost,shell=bash
  #    delimiter: "_"
```


## Metricsets [_metricsets_33]

The following metricsets are available:

* [server](/reference/metricbeat/metricbeat-metricset-graphite-server.md)


