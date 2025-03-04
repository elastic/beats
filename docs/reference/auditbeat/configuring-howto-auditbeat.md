---
navigation_title: "Configure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/configuring-howto-auditbeat.html
---

# Configure Auditbeat [configuring-howto-auditbeat]


::::{tip}
To get started quickly, read [Quick start: installation and configuration](/reference/auditbeat/auditbeat-installation-configuration.md).
::::


To configure Auditbeat, edit the configuration file. The default configuration file is called  `auditbeat.yml`. The location of the file varies by platform. To locate the file, see [Directory layout](/reference/auditbeat/directory-layout.md).

There’s also a full example configuration file called `auditbeat.reference.yml` that shows all non-deprecated options.

::::{tip}
See the [Config File Format](/reference/libbeat/config-file-format.md) for more about the structure of the config file.
::::


The following topics describe how to configure Auditbeat:

* [Modules](/reference/auditbeat/configuration-auditbeat.md)
* [General settings](/reference/auditbeat/configuration-general-options.md)
* [Project paths](/reference/auditbeat/configuration-path.md)
* [Config file reloading](/reference/auditbeat/auditbeat-configuration-reloading.md)
* [Output](/reference/auditbeat/configuring-output.md)
* [SSL](/reference/auditbeat/configuration-ssl.md)
* [Index lifecycle management (ILM)](/reference/auditbeat/ilm.md)
* [Elasticsearch index template](/reference/auditbeat/configuration-template.md)
* [{{kib}} endpoint](/reference/auditbeat/setup-kibana-endpoint.md)
* [Kibana dashboards](/reference/auditbeat/configuration-dashboards.md)
* [Processors](/reference/auditbeat/filtering-enhancing-data.md)
* [Internal queue](/reference/auditbeat/configuring-internal-queue.md)
* [Logging](/reference/auditbeat/configuration-logging.md)
* [HTTP endpoint](/reference/auditbeat/http-endpoint.md)
* [*Regular expression support*](/reference/auditbeat/regexp-support.md)
* [Instrumentation](/reference/auditbeat/configuration-instrumentation.md)
* [Feature flags](/reference/auditbeat/configuration-feature-flags.md)
* [*auditbeat.reference.yml*](/reference/auditbeat/auditbeat-reference-yml.md)

After changing configuration settings, you need to restart Auditbeat to pick up the changes.

