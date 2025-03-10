---
navigation_title: "Configure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/configuring-howto-heartbeat.html
---

# Configure Heartbeat [configuring-howto-heartbeat]


::::{tip}
To get started quickly, read [Quick start: installation and configuration](/reference/heartbeat/heartbeat-installation-configuration.md).
::::


To configure Heartbeat, edit the configuration file. The default configuration file is called  `heartbeat.yml`. The location of the file varies by platform. To locate the file, see [Directory layout](/reference/heartbeat/directory-layout.md).

There’s also a full example configuration file called `heartbeat.reference.yml` that shows all non-deprecated options.

::::{tip}
See the [Config File Format](/reference/libbeat/config-file-format.md) for more about the structure of the config file.
::::


The following topics describe how to configure Heartbeat:

* [Monitors](/reference/heartbeat/configuration-heartbeat-options.md)
* [Task scheduler](/reference/heartbeat/monitors-scheduler.md)
* [General settings](/reference/heartbeat/configuration-general-options.md)
* [Project paths](/reference/heartbeat/configuration-path.md)
* [Output](/reference/heartbeat/configuring-output.md)
* [SSL](/reference/heartbeat/configuration-ssl.md)
* [Index lifecycle management (ILM)](/reference/heartbeat/ilm.md)
* [Elasticsearch index template](/reference/heartbeat/configuration-template.md)
* [Processors](/reference/heartbeat/filtering-enhancing-data.md)
* [*Autodiscover*](/reference/heartbeat/configuration-autodiscover.md)
* [Internal queue](/reference/heartbeat/configuring-internal-queue.md)
* [Logging](/reference/heartbeat/configuration-logging.md)
* [HTTP endpoint](/reference/heartbeat/http-endpoint.md)
* [*Regular expression support*](/reference/heartbeat/regexp-support.md)
* [Instrumentation](/reference/heartbeat/configuration-instrumentation.md)
* [Feature flags](/reference/heartbeat/configuration-feature-flags.md)
* [*heartbeat.reference.yml*](/reference/heartbeat/heartbeat-reference-yml.md)

