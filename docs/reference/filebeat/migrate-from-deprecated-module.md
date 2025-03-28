---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/migrate-from-deprecated-module.html
---

# Migrating from a Deprecated Filebeat Module [migrate-from-deprecated-module]

If a Filebeat module has been deprecated, there are a few options available for a path forward:

1. Migrate to an Elastic integration, if available. The deprecation notice will link to an appropriate integration, if one exists.
2. [Migrate to Elastic Agent](docs-content://reference/fleet/migrate-from-beats-to-elastic-agent.md) for ingesting logs. If a specific integration for the vendor/product does not exist, then one of the custom integrations can be used for ingesting events. A [custom pipeline](docs-content://reference/fleet/data-streams-pipeline-tutorial.md) may also be attached to the integration for further processing.

    * [CEL Custom API](integration-docs://reference/cel/index.md) - Collect events from an API using CEL (Common Expression Language)
    * [Custom API](integration-docs://reference/httpjson/index.md) - Collect events from an API using the HTTPJSON input
    * [Custom Google Pub/Sub](integration-docs://reference/gcp_pubsub/index.md) - Collect events from Google Pub/Sub topics
    * [Custom HTTP Endpoint](integration-docs://reference/http_endpoint/index.md) - Collect events from a listening HTTP port
    * [Custom Journald](integration-docs://reference/journald/index.md) - Collect events from journald
    * [Custom Kafka](integration-docs://reference/kafka_log/index.md) - Collect events from a Kafka topic
    * [Custom Logs](integration-docs://reference/log/index.md) - Collect events from files
    * [Custom TCP](integration-docs://reference/tcp/index.md) - Collect events from a listening TCP port
    * [Custom UDP](integration-docs://reference/udp/index.md) - Collect events from a listening UDP port
    * [Custom Windows Event](integration-docs://reference/winlog/index.md) - Collect events from a Windows Event Log channel

3. Migrate to a different Filebeat module. In some cases, a Filebeat module may be superseded by a new module. The deprecation notice will link to an appropriate module, if one exists.
4. Use a custom Filebeat input, processors, and ingest pipeline (if necessary).

