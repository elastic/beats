---
navigation_title: "DNS"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-dns-options.html
---

# Capture DNS traffic [packetbeat-dns-options]


The `dns` section of the `packetbeat.yml` config file specifies configuration options for the DNS protocol. The DNS protocol supports processing DNS messages on TCP and UDP. Here is a sample configuration section for DNS:

```yaml
packetbeat.protocols:
- type: dns
  ports: [53]
  include_authorities: true
  include_additionals: true
```

## Configuration options [_configuration_options_3]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `include_authorities` [_include_authorities]

If this option is enabled, dns.authority fields (authority resource records) are added to DNS events. The default is false.


### `include_additionals` [_include_additionals]

If this option is enabled, dns.additionals fields (additional resource records) are added to DNS events. The default is false.



