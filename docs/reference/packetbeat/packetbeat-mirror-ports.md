---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-mirror-ports.html
---

# Packetbeat doesn't see any packets when using mirror ports [packetbeat-mirror-ports]

The interface needs to be set to promiscuous mode. Run the following command:

```sh
ip link set <device_name> promisc on
```

For example: `ip link set enp5s0f1 promisc on`

