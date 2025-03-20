---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/publishing-ls-fails-connection-reset-by-peer.html
---

# Publishing to Logstash fails with "connection reset by peer" message [publishing-ls-fails-connection-reset-by-peer]

Metricbeat requires a persistent TCP connection to {{ls}}. If a firewall interferes with the connection, you might see errors like this:

```shell
Failed to publish events caused by: write tcp ... write: connection reset by peer
```

To solve the problem:

* make sure the firewall is not closing connections between Metricbeat and {{ls}}, or
* set the `ttl` value in the [{{ls}} output](/reference/metricbeat/logstash-output.md) to a value that’s lower than the maximum time allowed by the firewall, and set `pipelining` to 0 (pipelining cannot be enabled when `ttl` is used).

