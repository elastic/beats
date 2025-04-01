---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/publishing-ls-fails-connection-reset-by-peer.html
---

# Publishing to Logstash fails with "connection reset by peer" message [publishing-ls-fails-connection-reset-by-peer]

Filebeat requires a persistent TCP connection to {{ls}}. If a firewall interferes with the connection, you might see errors like this:

```shell
Failed to publish events caused by: write tcp ... write: connection reset by peer
```

To solve the problem:

* make sure the firewall is not closing connections between Filebeat and {{ls}}, or
* set the `ttl` value in the [{{ls}} output](/reference/filebeat/logstash-output.md) to a value that’s lower than the maximum time allowed by the firewall, and set `pipelining` to 0 (pipelining cannot be enabled when `ttl` is used).

