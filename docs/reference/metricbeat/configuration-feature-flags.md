---
navigation_title: "Feature flags"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-feature-flags.html
---

# Configure feature flags [configuration-feature-flags]


The Feature Flags section of the `metricbeat.yml` config file contains settings in Metricbeat that are disabled by default. These may include experimental features, changes to behaviors within Metricbeat, or settings that could cause a breaking change. For example a setting that changes information included in events might be inconsistent with the naming pattern expected in your configured Metricbeat output.

To enable any of the settings listed on this page, change the associated `enabled` flag from `false` to `true`.

```yaml
features:
  mysetting:
    enabled: true
```


## Configuration options [_configuration_options_17]

You can specify the following options in the `features` section of the `metricbeat.yml` config file:


### `fqdn` [_fqdn]

Contains configuration for the FQDN reporting feature. When this feature is enabled, the fully-qualified domain name for the host is reported in the `host.name` field in events produced by Metricbeat.

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


#### `enabled` [_enabled_11]

Set to `true` to enable the FQDN reporting feature of Metricbeat. Defaults to `false`.

