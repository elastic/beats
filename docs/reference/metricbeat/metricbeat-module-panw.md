---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-panw.html
---

# Panw module [metricbeat-module-panw]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/panw/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The panw Metricbeat module uses the Palo Alto [pango](https://pkg.go.dev/github.com/PaloAltoNetworks/pango#section-documentation) package to extract metrics information from a firewall device via the XML API.


### Dashboards [_dashboards_3]


### Module-specific configuration notes [_module_specific_configuration_notes_15]

The panw module configuration requires the ip address of the target firewall device and an API Key generated from that firewall. It is assumed that network access to the firewall is available. All access by the panw module is read-only.

**Limitations** The current version of the module is configured to run against **exactly 1** firewall. Multiple firewalls will require multiple agent configurations. The module has also not been tested with Panorama, though it should work since it only relies on lower level Client.Op calls to send XML API commands to the server.

Required credentials for the `panw` module:

`host_ip`
:   IP address of the firewall - must be network accessible.

`apiKey`
:   An API Key generated via an XML API call to the firewall or via the management dashboard. This


## Metricsets [_metricsets_60]


### `bgp_peers` [_bgp_peers]

This metricset reports information on BGP Peers defined in the firewall.


### `certificates` [_certificates]

This metricset will capture certificates defined on the firewall including expiration dates.


### `fans` [_fans]

This metricset will collect information from hardware fans (RPMS) and will report if an alarm is active for a given fan.


### `filesystem` [_filesystem]

This metricset reports disk usage for filesystems defined on the device, based on df output.


### `globalprotect_sessions` [_globalprotect_sessions]

This metricset will collect metrics on current user sessions established on Global Protect gateways.


### `globalprotect_stats` [_globalprotect_stats]

This metricset reports the number of user per GlobalProtect gateway and totals across all gateways.


### `ha_interfaces` [_ha_interfaces]

This metricset will collect metrics from the device on High Availabilty configuration for interfaces.


### `licenses` [_licenses]

This metricset reports on licenses for sofware features with expiration dates.


### `logical` [_logical]

This metricset will collect metrics on logical interfaces in the device’s network.


### `power` [_power]

This metricset reports power usage and alarms.


### `system` [_system]

This metricset captures system informate such as uptime, user count, CPU, memory and swap: essentiallyl the first 5 lines of *top* output.


### `temperature` [_temperature]

This metricset reports temperature for various slots on the device and reports on alarm status.


### `tunnels` [_tunnels]

This metricset enumerates ipsec tunnels and their status.


### Example configuration [_example_configuration_52]

The Panw module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: panw
  metricsets: ["licenses"]
  enabled: false
  period: 10s
  hosts: ["localhost"]
```


### Metricsets [_metricsets_61]

The following metricsets are available:

* [interfaces](/reference/metricbeat/metricbeat-metricset-panw-interfaces.md)
* [routing](/reference/metricbeat/metricbeat-metricset-panw-routing.md)
* [system](/reference/metricbeat/metricbeat-metricset-panw-system.md)
* [vpn](/reference/metricbeat/metricbeat-metricset-panw-vpn.md)
