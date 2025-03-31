---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-zoom.html
---

# Zoom module [filebeat-module-zoom]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/zoom/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for Zoom webhook logs. The module creates an HTTP listener that accepts incoming webhooks from Zoom.

To configure Zoom to send webhooks to the filebeat module, please follow the [Zoom Documentation](https://developers.zoom.us/docs/api/rest/webhook-only-app).

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-zoom-module]

You can further refine the behavior of the `zoom` module by specifying [variable settings](#zoom-settings) in the `modules.d/zoom.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [zoom-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `zoom` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `zoom.webhook.var.paths` instead of `webhook.var.paths`.
::::



### `webhook` fileset settings [_webhook_fileset_settings]

When a webhook integration is created on Zoom, you can create a custom header to verify webhook events. See [Custom Header](https://developers.zoom.us/docs/api/rest/webhook-reference/#custom-header) for more information about this process. This is configured with the `secret.header` and `secret.value` settings as shown below.

On the other hand, Zoom also requires webhook validation for created or modified webhooks after October, 2022. This follows a challenge-response check (CRC) algorithm which is configured with the `crc.enabled` and `crc.secret` settings. Learn more about it at [Validate your webhook endpoint](https://developers.zoom.us/docs/api/rest/webhook-reference/#validate-your-webhook-endpoint).

Example config:

```yaml
- module: zoom
  webhook:
    enabled: true
    var.input: http_endpoint
    var.listen_address: 0.0.0.0
    var.listen_port: 8080
    var.secret.header: x-my-custom-key
    var.secret.value: my-custom-value
    var.crc.enabled: true
    var.crc.secret: ZOOMSECRETTOKEN
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.listen_address`**
:   The IP address of the interface the module should listen on. Also supports 0.0.0.0 to listen on all interfaces.

**`var.listen_port`**
:   The port the module should be listening on.

**`var.ssl`**
:   Configuration options for SSL parameters like the SSL certificate and CA to use for the HTTP(s) listener See [SSL](/reference/filebeat/configuration-ssl.md) for more information.


## Fields [_fields_57]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-zoom.md) section.
