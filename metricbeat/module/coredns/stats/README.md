# CoreDNS Stats

## Version history

- May 2019, `v1.5.0`

## Resources

- https://github.com/coredns/coredns/tree/master/plugin/metrics
- https://coredns.io/manual/configuration/

## Setup environment for manual tests

Write this contents to `corefile`

```
# Zone1
domain.elastic:1053 {
    log
    errors
    auto
    reload 10s
    cache 4

    prometheus :9153

    hosts {
        127.0.0.1   my.domain.elastic
        192.168.0.1   theirs.domain.elastic
        fallthrough
    }
}

# Zone2
.:1053 {
    log
    errors
    prometheus :9153
    cache 4

    forward . 8.8.8.8 8.8.4.4
}
```

It creates 2 zones listening on port 1053, prometheus metrics can be gathered at port 9153
Requests for `my.domain.elastic` and `theirs.domain.elastic` will be resolved locally
Any other request will be forwarded to google's DNSs.
Cache plugin is activated

For manual testing you can open a number of terminals and use `watch` with any of these commands:

```
dig @localhost -p 1053 TXT apache.org

dig @localhost -p 1053 A google.com

dig @localhost -p 1053 MX google.com

dig @localhost -p 1053 A my.domain.elastic

dig @localhost -p 1053 A theirs.domain.elastic +tcp

```

Metrics can be manually retrieved using

```
curl localhost:9153/metrics
```
