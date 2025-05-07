---
navigation_title: "ICMP"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-icmp-options.html
---

# Capture ICMP traffic [packetbeat-icmp-options]


The `icmp` section of the `packetbeat.yml` config file specifies options for the ICMP protocol. Here is a sample configuration section for ICMP:

```yaml
packetbeat.protocols:

- type: icmp
  enabled: true
```

## Configuration options [_configuration_options_2]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `enabled` [_enabled_3]

The ICMP protocol can be enabled/disabled via this option. The default is true.

If enabled Packetbeat will generate the following BPF filter: `"icmp or icmp6"`.



