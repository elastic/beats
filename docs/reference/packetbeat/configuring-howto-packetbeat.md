---
navigation_title: "Configure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuring-howto-packetbeat.html
---

# Configure Packetbeat [configuring-howto-packetbeat]


::::{tip}
To get started quickly, read [Quick start: installation and configuration](/reference/packetbeat/packetbeat-installation-configuration.md).
::::


To configure Packetbeat, edit the configuration file. The default configuration file is called  `packetbeat.yml`. The location of the file varies by platform. To locate the file, see [Directory layout](/reference/packetbeat/directory-layout.md).

There’s also a full example configuration file called `packetbeat.reference.yml` that shows all non-deprecated options.

::::{tip}
See the [Config File Format](/reference/libbeat/config-file-format.md) for more about the structure of the config file.
::::


The following topics describe how to configure Packetbeat:

* [Network flows](/reference/packetbeat/configuration-flows.md)
* [Protocols](/reference/packetbeat/configuration-protocols.md)
* [Processes](/reference/packetbeat/configuration-processes.md)
* [General settings](/reference/packetbeat/configuration-general-options.md)
* [Project paths](/reference/packetbeat/configuration-path.md)
* [Traffic sniffing](/reference/packetbeat/configuration-interfaces.md)
* [Output](/reference/packetbeat/configuring-output.md)
* [SSL](/reference/packetbeat/configuration-ssl.md)
* [Index lifecycle management (ILM)](/reference/packetbeat/ilm.md)
* [Elasticsearch index template](/reference/packetbeat/configuration-template.md)
* [{{kib}} endpoint](/reference/packetbeat/setup-kibana-endpoint.md)
* [Kibana dashboards](/reference/packetbeat/configuration-dashboards.md)
* [Processors](/reference/packetbeat/filtering-enhancing-data.md)
* [Internal queue](/reference/packetbeat/configuring-internal-queue.md)
* [Logging](/reference/packetbeat/configuration-logging.md)
* [HTTP endpoint](/reference/packetbeat/http-endpoint.md)
* [Instrumentation](/reference/packetbeat/configuration-instrumentation.md)
* [Feature flags](/reference/packetbeat/configuration-feature-flags.md)
* [*packetbeat.reference.yml*](/reference/packetbeat/packetbeat-reference-yml.md)

