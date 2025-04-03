---
navigation_title: "Feature flags"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-feature-flags.html
---

# Configure feature flags [configuration-feature-flags]


The Feature Flags section of the `packetbeat.yml` config file contains settings in Packetbeat that are disabled by default. These may include experimental features, changes to behaviors within Packetbeat, or settings that could cause a breaking change. For example a setting that changes information included in events might be inconsistent with the naming pattern expected in your configured Packetbeat output.

To enable any of the settings listed on this page, change the associated `enabled` flag from `false` to `true`.

```yaml
features:
  mysetting:
    enabled: true
```


## Configuration options [_configuration_options_30]

You can specify the following options in the `features` section of the `packetbeat.yml` config file:


### `fqdn` [_fqdn]

Contains configuration for the FQDN reporting feature. When this feature is enabled, the fully-qualified domain name for the host is reported in the `host.name` field in events produced by Packetbeat.

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


For FQDN reporting to work as expected, the hostname of the current host must either:

* Have a CNAME entry defined in DNS.
* Have one of its corresponding IP addresses respond successfully to a reverse DNS lookup.

If neither pre-requisite is satisfied, `host.name` continues to report the hostname of the current host as if the FQDN feature flag were not enabled.

Example configuration:

```yaml
features:
  fqdn:
    enabled: true
```


#### `enabled` [_enabled_13]

Set to `true` to enable the FQDN reporting feature of Packetbeat. Defaults to `false`.

