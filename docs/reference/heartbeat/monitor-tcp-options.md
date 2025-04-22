---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/monitor-tcp-options.html
---

# TCP options [monitor-tcp-options]

Also see [Common monitor options](/reference/heartbeat/monitor-options.md).

The options described here configure Heartbeat to connect via TCP and optionally verify the endpoint by sending and/or receiving a custom payload.

Example configuration:

```yaml
- type: tcp
  id: my-host-services
  name: My Host Services
  hosts: ["myhost"]
  ports: [80, 9200, 5044]
  schedule: '@every 5s'
```


## `hosts` [monitor-tcp-hosts]

A list of hosts to ping. The entries in the list can be:

* A plain host name, such as `localhost`, or an IP address. If you specify this option, you must also specify a value for [`ports`](#monitor-tcp-ports).  If the monitor is [configured to use SSL](/reference/heartbeat/configuration-ssl.md), Heartbeat establishes an SSL/TLS-based connection. Otherwise, it establishes a plain TCP connection.
* A hostname and port, such as `localhost:12345`. Heartbeat connects to the port on the specified host. If the monitor is [configured to use SSL](/reference/heartbeat/configuration-ssl.md), Heartbeat establishes an SSL/TLS-based connection. Otherwise, it establishes a TCP connection.
* A full URL using the syntax `scheme://<host>:[port]`, where:

    * `scheme` is one of `tcp`, `plain`, `ssl` or `tls`. If `tcp` or `plain` is specified, Heartbeat establishes a TCP connection even if the monitor is configured to use SSL. If `tls` or `ssl` is specified, Heartbeat establishes an SSL connection. However, if the monitor is not configured to use SSL, the system defaults are used (currently not supported on Windows).
    * `host` is the hostname.
    * `port` is the port number. If `port` is missing in the URL, the [`ports`](#monitor-tcp-ports) setting is required.



## `ports` [monitor-tcp-ports]

A list of ports to ping if the host specified in [`hosts`](#monitor-tcp-hosts) does not contain a port number. It is generally preferable to use a single value here, since each port will be monitored using a separate `id`, with the given `id` value, used as a prefix in the Heartbeat data, and the configured `name` shared across events sent via this check.


## `check` [monitor-tcp-check]

An optional payload string to send to the remote host and the expected answer. If no payload is specified, the endpoint is assumed to be available if the connection attempt was successful. If `send` is specified without `receive`, any response is accepted as OK. If `receive` is specified without `send`, no payload is sent, but the client expects to receive a payload in the form of a "hello message" or "banner" on connect.

Example configuration:

```yaml
- type: tcp
  id: echo-service
  name: Echo Service
  hosts: ["myhost"]
  ports: [7]
  check.send: 'Hello World'
  check.receive: 'Hello World'
  schedule: '@every 5s'
```


## `proxy_url` [monitor-tcp-proxy-url]

The URL of the SOCKS5 proxy to use when connecting to the server. The value must be a URL with a scheme of socks5://.

If the SOCKS5 proxy server requires client authentication, then a username and password can be embedded in the URL as shown in the example.

```yaml
  proxy_url: socks5://user:password@socks5-proxy:2233
```

When using a proxy, hostnames are resolved on the proxy server instead of on the client. You can change this behavior by setting the `proxy_use_local_resolver` option.


## `proxy_use_local_resolver` [monitor-tcp-proxy-use-local-resolver]

A Boolean value that determines whether hostnames are resolved locally instead of being resolved on the proxy server. The default value is false, which means that name resolution occurs on the proxy server.


## `ssl` [monitor-tcp-tls-ssl]

The TLS/SSL connection settings.  If the monitor is [configured to use SSL](/reference/heartbeat/configuration-ssl.md), it will attempt an SSL handshake. If `check` is not configured, the monitor will only check to see if it can establish an SSL/TLS connection. This check can fail either at TCP level or during certificate validation.

Example configuration:

```yaml
- type: tcp
  id: tls-mail
  name: TLS Mail
  hosts: ["mail.example.net"]
  ports: [465]
  schedule: '@every 5s'
  ssl:
    certificate_authorities: ['/etc/ca.crt']
    supported_protocols: ["TLSv1.0", "TLSv1.1", "TLSv1.2"]
```

Also see [SSL](/reference/heartbeat/configuration-ssl.md) for a full description of the `ssl` options.

