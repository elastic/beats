---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/monitor-http-options.html
---

# HTTP options [monitor-http-options]

Also see [Common monitor options](/reference/heartbeat/monitor-options.md).

The options described here configure Heartbeat to connect via HTTP and optionally verify that the host returns the expected response.

Example configuration:

```yaml
- type: http
  id: myhost
  name: My HTTP Host
  schedule: '@every 5s'
  hosts: ["http://myhost:80"]
```


### `hosts` [monitor-http-urls]

A list of URLs to ping.


### `max_redirects` [monitor-http-max-redirects]

The total number of redirections Heartbeat will follow. Defaults to 0, meaning heartbeat will not follow redirects, but will report the status of the redirect. If set to a number greater than 0 heartbeat will follow that number of redirects.

When this option is set to a value greater than zero the `monitor.ip` field will no longer be reported, as multiple DNS requests across multiple IPs may return multiple IPs. Fine grained network timing data will also not be recorded, as with redirects that data will span multiple requests. Specifically the fields `http.rtt.content.us`, `http.rtt.response_header.us`, `http.rtt.total.us`, `http.rtt.validate.us`, `http.rtt.write_request.us` and `dns.rtt.us` will be omitted.


### `proxy_url` [monitor-http-proxy-url]

The HTTP proxy URL. This setting is optional. Example `http://proxy.mydomain.com:3128`


### `proxy_headers` [monitor-http-proxy-headers]

Additional headers to send to proxies during CONNECT requests.


### `username` [monitor-http-username]

The username for authenticating with the server. The credentials are passed with the request. This setting is optional.

You need to specify credentials when your `check.response` settings require it. For example, you can check for a 403 response (`check.response.status: [403]`) without setting credentials.


### `password` [monitor-http-password]

The password for authenticating with the server. This setting is optional.


### `ssl` [monitor-http-tls-ssl]

The TLS/SSL connection settings for use with the HTTPS endpoint. If you don’t specify settings, the system defaults are used.

Example configuration:

```yaml
- type: http
  id: my-http-service
  name: My HTTP Service
  hosts: ["https://myhost:443"]
  schedule: '@every 5s'
  ssl:
    certificate_authorities: ['/etc/ca.crt']
    supported_protocols: ["TLSv1.0", "TLSv1.1", "TLSv1.2"]
```

Also see [SSL](/reference/heartbeat/configuration-ssl.md) for a full description of the `ssl` options.


## `headers` [monitor-http-headers]

Controls the indexing of the HTTP response headers `http.response.body.headers` field.

On by default. Set `response.include_headers` to `false` to disable.


## `response` [monitor-http-response]

Controls the indexing of the HTTP response body contents to the `http.response.body.contents` field.

Set `response.include_body` to one of the options listed below.

**`on_error`**
:   Include the body if an error is encountered during the check. This is the default.

**`never`**
:   Never include the body.

**`always`**
:   Always include the body with checks.

Set `response.include_body_max_bytes` to control the maximum size of the stored body contents. Defaults to 1024 bytes.


### `check` [monitor-http-check]

An optional `request` to send to the remote host and the expected `response`.

Example configuration:

```yaml
- type: http
  id: my-http-host
  name: My HTTP Service
  hosts: ["http://myhost:80"]
  check.request.method: HEAD
  check.response.status: [200]
  schedule: '@every 5s'
```

Under `check.request`, specify these options:

**`method`**
:   The HTTP method to use. Valid values are `"HEAD"`, `"GET"`, `"POST"`, `"PUT"`, `"DELETE"`, `"CONNECT"`, `"TRACE"` and `"OPTIONS"`.

**`headers`**
:   A dictionary of additional HTTP headers to send. By default heartbeat will set the *User-Agent* header to identify itself.

**`body`**
:   Optional request body content.

Example configuration: This monitor POSTs an `x-www-form-urlencoded` string to the endpoint `/demo/add`

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  urls: ["http://localhost:8080/demo/add"]
  check.request:
    method: POST
    headers:
      'Content-Type': 'application/x-www-form-urlencoded'
    # urlencode the body:
    body: "name=first&email=someemail%40someemailprovider.com"
  check.response:
    status: [200]
    body:
      - Saved
      - saved
```

Under `check.response`, specify these options:

**`status`**
:   A list of expected status codes. 4xx and 5xx codes are considered `down` by default. Other codes are considered `up`.

**`headers`**
:   The required response headers.

**`body`**
:   A list of regular expressions to match the body output. Only a single expression needs to match. HTTP response bodies of up to 100MiB are supported.

Example configuration: This monitor examines the response body for the strings `saved` or `Saved` and expects 200 or 201 status codes

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  urls: ["http://localhost:8080/demo/add"]
  check.request:
    method: POST
    headers:
      'Content-Type': 'application/x-www-form-urlencoded'
    # urlencode the body:
    body: "name=first&email=someemail%40someemailprovider.com"
  check.response:
    status: [200, 201]
    body:
      - Saved
      - saved
```

Under `check.response.body`, specify these options: **`positive`**:: This option has the same behavior as given a list of regular expressions under `check.response.body`. **`negative`**:: A list of regular expressions to match the the body output negatively. Return match failed if single expression matches. HTTP response bodies of up to 100MiB are supported.

Example configuration: This monitor examines the response body for the strings *foo* or *Foo*

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  urls: ["http://localhost:8080/demo/add"]
  check.request:
    method: POST
    headers:
      'Content-Type': 'application/x-www-form-urlencoded'
    # urlencode the body:
    body: "name=first&email=someemail%40someemailprovider.com"
  check.response:
    body:
      positive:
        - foo
        - Foo
```

Example configuration: This monitor examines match successfully if there is no *bar* or *Bar* at all, examines match failed if there is *bar* or *Bar* in the response body

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  urls: ["http://localhost:8080/demo/add"]
  check.request:
    method: POST
    headers:
      'Content-Type': 'application/x-www-form-urlencoded'
    # urlencode the body:
    body: "name=first&email=someemail%40someemailprovider.com"
  check.response:
    status: [200, 201]
    body:
      negative:
        - bar
        - Bar
```

Example configuration: This monitor examines match successfully only when *foo* or *Foo* in body AND no *bar* or *Bar* in body

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  urls: ["http://localhost:8080/demo/add"]
  check.response:
    status: [200, 201]
    body:
      positive:
        - foo
        - Foo
      negative:
        - bar
        - Bar
```

**`json`**
:   A list of expressions or [condition](/reference/heartbeat/defining-processors.md#conditions) statements (now deprecated) executed against the body when parsed as JSON. Body sizes must be less than or equal to 100 MiB.

The following configuration shows how to check the response using [gval](https://github.com/PaesslerAG/gval/blob/master/README.md) expressions when the body contains JSON:

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  hosts: ["https://localhost:9200/_/nodes/stats"]
  username: elastic
  password: changeme
  check.response:
    status: [200]
    json:
      - description: check status
        expression: 'foo.bar == "myValue"'
```

Expressions can be much more complex than simple equality. They can also use [jsonpath](https://goessner.net/articles/JsonPath/) syntax. Note that strings must be double quoted with `"` rather than single quoted with `'` in the `gval` variant of jsonpath. Please note that jsonpath sub-expressions must start with `$.`, for instance `'$.nodes[?(@.name=="myname")] != []'` will check that the `nodes` map has at least one value with the name *myname*.

When working with responses that are returned in the form of a JSON array at the root rather than an object jsonpath can be used as well. As an example `$.[0].foo == "bar"` tests that the first item in the response has an attribute `foo` that has the value "bar".

JSON bodies can also be checked via the now deprecated `condition` option, which is not as powerful as `expression`. The following configuration shows how to check the response using a `condition` statement when the body contains JSON:

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  hosts: ["https://myhost:80"]
  check.request:
    method: GET
    headers:
      'X-API-Key': '12345-mykey-67890'
  check.response:
    status: [200]
    json:
      - description: check status
        condition:
          equals:
            status: ok
```

The following configuration shows how to check the response for multiple regex patterns:

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  hosts: ["https://myhost:80"]
  check.request:
    method: GET
    headers:
      'X-API-Key': '12345-mykey-67890'
  check.response:
    status: [200]
    body:
      - hello
      - world
```

The following configuration shows how to check the response with a multiline regex:

```yaml
- type: http
  id: demo-service
  name: Demo Service
  schedule: '@every 5s'
  hosts: ["https://myhost:80"]
  check.request:
    method: GET
    headers:
      'X-API-Key': '12345-mykey-67890'
  check.response:
    status: [200]
    body: '(?s)first.*second.*third'
```


## Run Once Mode (Experimental) [run-once-mode]

You can configure Heartbeat run monitors exactly once then exit, bypassing the scheduler. This is referred to as running Heartbeat in "run once" mode by setting `heartbeat.run_once: true`. All Heartbeat monitors will ignore their schedules and run exactly once at startup. This is an experimental feature and is subject to change.

Note, the `schedule` field is still required and is used by Heartbeat to set the expectation around when the next run will occur. That duration is encoded in the `monitor.timespan` field in the Heartbeat output.

```yaml
# heartbeat.yml
heartbeat.run_once: true
heartbeat.monitors:
# your monitor config here...
```


## Publish timeout (Experimental) [publish-timeout]

You can configure Heartbeat to exit after an elapsed timeout if unable to publish pending events. This is an experimental feature and is subject to change.

Note, the `heartbeat.run_once` flag is required for `publish_timeout` to take effect.

```yaml
# heartbeat.yml
heartbeat.publish_timeout: 30s
heartbeat.run_once: true
heartbeat.monitors:
# your monitor config here...
```

