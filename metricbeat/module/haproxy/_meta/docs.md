:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/haproxy/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module collects stats from [HAProxy](http://www.haproxy.org/). It supports collection from TCP sockets, UNIX sockets, or HTTP with or without basic authentication.

Metricbeat can collect two metricsets from HAProxy: `info` and `stat`. `info` is not available when using the stats page.


## Configure HAProxy to collect stats [_configure_haproxy_to_collect_stats]

Before you can use Metricbeat to collect stats, you must enable the stats module in HAProxy. You can do this a couple of ways: configure HAProxy to report stats via a TCP or UNIX socket, or enable the stats page.


### TCP socket [_tcp_socket]

To enable stats reporting via any local IP on port 14567, add the following line to the `global` or `default` section of the HAProxy config:

```shell
 stats socket 127.0.0.1:14567
```

::::{note}
You should use an internal private IP, or secure this with a firewall rule, so that only designated hosts can access this data.
::::



### UNIX socket [_unix_socket]

To enable stats reporting via a UNIX socket, add the following line to the `global` or `default` section of the HAProxy config:

```shell
 stats socket /path/to/haproxy.sock mode 660 level admin
```


### Stats page [_stats_page]

To enable the HAProxy stats page, add the following lines to the HAProxy config, then restart HAProxy. The stats page in this example will be available to any IP on port 14567 after authentication.

```text
 listen stats
   bind 0.0.0.0:14567
   stats enable
   stats uri /stats
   stats auth admin:admin
```


## Compatibility [_compatibility_22]

The HAProxy metricsets are tested with HAProxy versions from 1.6 to 1.8.
