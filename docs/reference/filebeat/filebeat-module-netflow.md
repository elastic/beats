---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-netflow.html
---

# NetFlow module [filebeat-module-netflow]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/netflow/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for receiving NetFlow and IPFIX flow records over UDP. This input supports NetFlow versions 1, 5, 6, 7, 8 and 9, as well as IPFIX. For NetFlow versions older than 9, fields are mapped automatically to NetFlow v9.

This module wraps the [netflow input](/reference/filebeat/filebeat-input-netflow.md) to enrich the flow records with geolocation information about the IP endpoints by using an {{es}} ingest pipeline.

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-netflow-module]

You can further refine the behavior of the `netflow` module by specifying [variable settings](#netflow-settings) in the `modules.d/netflow.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [netflow-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `netflow` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `netflow.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_9]

The fileset is by default configured to listen for UDP traffic on `localhost:2055`. For most uses cases you will want to set the `netflow_host` variable to allow the input bind to all interfaces so that it can receive traffic from network devices.

```yaml
- module: netflow
  log:
    enabled: true
    var:
      netflow_host: 0.0.0.0
      netflow_port: 2055
```

`var.netflow_host`
:   Address to bind to. Defaults to `localhost`.

`var.netflow_port`
:   Port to listen on. Defaults to `2055`.

`var.max_message_size`
:   The maximum size of the message received over UDP. The default is `10KiB`.

`var.read_buffer`
:   The size of the read buffer on the UDP socket.

`var.timeout`
:   The read and write timeout for socket operations.

`var.expiration_timeout`
:   The time before an idle session or unused template is expired. Only applicable to v9 and IPFIX protocols. A value of zero disables expiration.

`var.queue_size`
:   The maximum number of packets that can be queued for processing. Use this setting to avoid packet-loss when dealing with occasional bursts of traffic.

`var.custom_definitions`
:   A list of paths to field definitions YAML files. These allow to update the NetFlow/IPFIX fields with vendor extensions and to override existing fields. See [netflow input](/reference/filebeat/filebeat-input-netflow.md) for details.

`var.detect_sequence_reset`
:   Flag controlling whether Filebeat should monitor sequence numbers in the Netflow packets to detect an Exporting Process reset. See [netflow input](/reference/filebeat/filebeat-input-netflow.md) for details.

`var.internal_networks`
:   A list of CIDR ranges describing the IP addresses that you consider internal. This is used in determining the values of `source.locality`, `destination.locality`, and `flow.locality`. The values can be either a CIDR value or one of the named ranges supported by the [`network`](/reference/filebeat/defining-processors.md#condition-network) condition. The default value is `[private]` which classifies RFC 1918 (IPv4) and RFC 4193 (IPv6) addresses as internal.

**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Defaults to `[forwarded]`.


## Fields [_fields_36]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-netflow.md) section.
