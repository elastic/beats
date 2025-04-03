---
navigation_title: "Protocols"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-protocols.html
---

# Configure which transaction protocols to monitor [configuration-protocols]


The `packetbeat.protocols` section of the `packetbeat.yml` config file contains configuration options for each supported protocol, including common options like `enabled`, `ports`, `send_request`, `send_response`, and options that are protocol-specific.

Currently, Packetbeat supports the following protocols:

* ICMP (v4 and v6)
* DHCP (v4)
* DNS
* HTTP
* AMQP 0.9.1
* Cassandra
* Mysql
* PostgreSQL
* Redis
* Thrift-RPC
* MongoDB
* Memcache
* NFS
* TLS
* SIP/SDP (beta)

Example configuration:

```yaml
packetbeat.protocols:

- type: icmp
  enabled: true

- type: dhcpv4
  ports: [67, 68]

- type: dns
  ports: [53]

- type: http
  ports: [80, 8080, 8000, 5000, 8002]

- type: amqp
  ports: [5672]

- type: cassandra
  ports: [9042]

- type: memcache
  ports: [11211]

- type: mysql
  ports: [3306,3307]

- type: redis
  ports: [6379]

- type: pgsql
  ports: [5432]

- type: thrift
  ports: [9090]

- type: tls
  ports: [443, 993, 995, 5223, 8443, 8883, 9243]
```














