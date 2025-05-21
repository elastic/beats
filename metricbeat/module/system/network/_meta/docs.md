The System `network` metricset provides network IO metrics collected from the operating system. One event is created for each network interface.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* Windows


## Configuration [_configuration_11]

**`interfaces`**
:   By default metrics are reported from all network interfaces. To select which interfaces metrics are reported from, use the `interfaces` configuration option. The value must be an array of interface names. For example:

```yaml
metricbeat.modules:
- module: system
  metricsets: [network]
  interfaces: [eth0]
```

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
