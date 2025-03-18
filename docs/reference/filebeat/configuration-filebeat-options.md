---
navigation_title: "Inputs"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-filebeat-options.html
---

# Configure inputs [configuration-filebeat-options]


::::{tip}
[Filebeat modules](/reference/filebeat/filebeat-modules-overview.md) provide the fastest getting started experience for common log formats. See [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md) to learn how to get started.
::::


To configure Filebeat manually (instead of using [modules](/reference/filebeat/filebeat-modules-overview.md)), you specify a list of inputs in the `filebeat.inputs` section of the `filebeat.yml`. Inputs specify how Filebeat locates and processes input data.

The list is a [YAML](http://yaml.org/) array, so each input begins with a dash (`-`). You can specify multiple inputs, and you can specify the same input type more than once. For example:

```yaml
filebeat.inputs:
- type: filestream
  id: my-filestream-id <1>
  paths:
    - /var/log/system.log
    - /var/log/wifi.log
- type: filestream
  id: apache-filestream-id
  paths:
    - "/var/log/apache2/*"
  fields:
    apache: true
  fields_under_root: true
```

1. Each filestream input must have a unique ID to allow tracking the state of files.


For the most basic configuration, define a single input with a single path. For example:

```yaml
filebeat.inputs:
- type: filestream
  id: my-filestream-id
  paths:
    - /var/log/*.log
```

The input in this example harvests all files in the path `/var/log/*.log`, which means that Filebeat will harvest all files in the directory `/var/log/` that end with `.log`. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here.

To fetch all files from a predefined level of subdirectories, use this pattern: `/var/log/*/*.log`. This fetches all `.log` files from the subfolders of `/var/log`. It does not fetch log files from the `/var/log` folder itself. Currently it is not possible to recursively fetch all files in all subdirectories of a directory.


## Input types [filebeat-input-types]

You can configure Filebeat to use the following inputs:

* [AWS CloudWatch](/reference/filebeat/filebeat-input-aws-cloudwatch.md)
* [AWS S3](/reference/filebeat/filebeat-input-aws-s3.md)
* [Azure Event Hub](/reference/filebeat/filebeat-input-azure-eventhub.md)
* [Azure Blob Storage](/reference/filebeat/filebeat-input-azure-blob-storage.md)
* [Benchmark](/reference/filebeat/filebeat-input-benchmark.md)
* [CEL](/reference/filebeat/filebeat-input-cel.md)
* [Cloud Foundry](/reference/filebeat/filebeat-input-cloudfoundry.md)
* [CometD](/reference/filebeat/filebeat-input-cometd.md)
* [Container](/reference/filebeat/filebeat-input-container.md)
* [Entity Analytics](/reference/filebeat/filebeat-input-entity-analytics.md)
* [ETW](/reference/filebeat/filebeat-input-etw.md)
* [filestream](/reference/filebeat/filebeat-input-filestream.md)
* [GCP Pub/Sub](/reference/filebeat/filebeat-input-gcp-pubsub.md)
* [Google Cloud Storage](/reference/filebeat/filebeat-input-gcs.md)
* [HTTP Endpoint](/reference/filebeat/filebeat-input-http_endpoint.md)
* [HTTP JSON](/reference/filebeat/filebeat-input-httpjson.md)
* [journald](/reference/filebeat/filebeat-input-journald.md)
* [Kafka](/reference/filebeat/filebeat-input-kafka.md)
* [Log](/reference/filebeat/filebeat-input-log.md) (deprecated in 7.16.0, use [filestream](/reference/filebeat/filebeat-input-filestream.md))
* [MQTT](/reference/filebeat/filebeat-input-mqtt.md)
* [NetFlow](/reference/filebeat/filebeat-input-netflow.md)
* [Office 365 Management Activity API](/reference/filebeat/filebeat-input-o365audit.md)
* [Redis](/reference/filebeat/filebeat-input-redis.md)
* [Salesforce](/reference/filebeat/filebeat-input-salesforce.md)
* [Stdin](/reference/filebeat/filebeat-input-stdin.md)
* [Streaming](/reference/filebeat/filebeat-input-streaming.md)
* [Syslog](/reference/filebeat/filebeat-input-syslog.md)
* [TCP](/reference/filebeat/filebeat-input-tcp.md)
* [UDP](/reference/filebeat/filebeat-input-udp.md)
* [Unified Logs](/reference/filebeat/filebeat-input-unifiedlogs.md)
* [Unix](/reference/filebeat/filebeat-input-unix.md)
* [winlog](/reference/filebeat/filebeat-input-winlog.md)
