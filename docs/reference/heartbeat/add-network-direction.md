---
navigation_title: "add_network_direction"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/add-network-direction.html
---

# Add network direction [add-network-direction]


The `add_network_direction` processor attempts to compute the perimeter-based network direction given an a source and destination ip address and list of internal networks. The key `internal_networks` can contain either CIDR blocks or a list of special values enumerated in the network section of [Conditions](/reference/heartbeat/defining-processors.md#conditions).

```yaml
processors:
  - add_network_direction:
      source: source.ip
      destination: destination.ip
      target: network.direction
      internal_networks: [ private ]
```

See [Conditions](/reference/heartbeat/defining-processors.md#conditions) for a list of supported conditions.

