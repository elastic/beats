---
navigation_title: "Configure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuring-howto-filebeat.html
---

# Configure Filebeat [configuring-howto-filebeat]


::::{tip}
To get started quickly, read [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md).
::::


To configure Filebeat, edit the configuration file. The default configuration file is called  `filebeat.yml`. The location of the file varies by platform. To locate the file, see [Directory layout](/reference/filebeat/directory-layout.md).

There’s also a full example configuration file called `filebeat.reference.yml` that shows all non-deprecated options.

::::{tip}
See the [Config File Format](/reference/libbeat/config-file-format.md) for more about the structure of the config file.
::::


The following topics describe how to configure Filebeat:

* [Inputs](/reference/filebeat/configuration-filebeat-options.md)
* [Modules](/reference/filebeat/configuration-filebeat-modules.md)
* [General settings](/reference/filebeat/configuration-general-options.md)
* [Project paths](/reference/filebeat/configuration-path.md)
* [Config file loading](/reference/filebeat/filebeat-configuration-reloading.md)
* [Output](/reference/filebeat/configuring-output.md)
* [SSL](/reference/filebeat/configuration-ssl.md)
* [Index lifecycle management (ILM)](/reference/filebeat/ilm.md)
* [Elasticsearch index template](/reference/filebeat/configuration-template.md)
* [{{kib}} endpoint](/reference/filebeat/setup-kibana-endpoint.md)
* [Kibana dashboards](/reference/filebeat/configuration-dashboards.md)
* [Processors](/reference/filebeat/filtering-enhancing-data.md)
* [*Autodiscover*](/reference/filebeat/configuration-autodiscover.md)
* [Internal queue](/reference/filebeat/configuring-internal-queue.md)
* [Logging](/reference/filebeat/configuration-logging.md)
* [HTTP endpoint](/reference/filebeat/http-endpoint.md)
* [Regular expression support](/reference/filebeat/regexp-support.md)
* [Instrumentation](/reference/filebeat/configuration-instrumentation.md)
* [Feature flags](/reference/filebeat/configuration-feature-flags.md)
* [*filebeat.reference.yml*](/reference/filebeat/filebeat-reference-yml.md)

