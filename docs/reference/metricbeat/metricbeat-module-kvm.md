---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-kvm.html
---

# KVM module [metricbeat-module-kvm]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the kvm module.


## Example configuration [_example_configuration_38]

The KVM module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: kvm
  metricsets: ["dommemstat", "status"]
  enabled: true
  period: 10s
  hosts: ["unix:///var/run/libvirt/libvirt-sock"]
  # For remote hosts, setup network access in libvirtd.conf
  # and use the tcp scheme:
  # hosts: [ "tcp://<host>:16509" ]

  # Timeout to connect to Libvirt server
  #timeout: 1s
```


## Metricsets [_metricsets_44]

The following metricsets are available:

* [dommemstat](/reference/metricbeat/metricbeat-metricset-kvm-dommemstat.md)
* [status](/reference/metricbeat/metricbeat-metricset-kvm-status.md)



