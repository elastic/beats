---
navigation_title: "HTTP"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-http-options.html
---

# Capture HTTP traffic [packetbeat-http-options]


The HTTP protocol has several specific configuration options. Here is a sample configuration for the `http` section of the `packetbeat.yml` config file:

```yaml
packetbeat.protocols:
- type: http
  ports: [80, 8080, 8000, 5000, 8002]
  hide_keywords: ["pass", "password", "passwd"]
  send_headers: ["User-Agent", "Cookie", "Set-Cookie"]
  split_cookie: true
  real_ip_header: "X-Forwarded-For"
```

## Configuration options [_configuration_options_4]

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `hide_keywords` [_hide_keywords]

A list of query parameters that Packetbeat will automatically censor in the transactions that it saves. The values associated with these parameters are replaced by `'xxxxx'`. By default, no changes are made to the HTTP messages.

Packetbeat has this option because, unlike SQL traffic, which typically only contains the hashes of the passwords, HTTP traffic may contain sensitive data. To reduce security risks, you can configure this option to avoid sending the contents of certain HTTP POST parameters.

::::{warning}
This option replaces query parameters from GET requests and top-level parameters from POST requests. If sensitive data is encoded inside a parameter that you don’t specify here, Packetbeat cannot censor it. Also, note that if you configure Packetbeat to save the raw request and response fields (see the [`send_request`](/reference/packetbeat/common-protocol-options.md#send-request-option) and the [`send_response`](/reference/packetbeat/common-protocol-options.md#send-response-option) options), sensitive data may be present in those fields.
::::



### `redact_authorization` [_redact_authorization]

When this option is enabled, Packetbeat obscures the value of `Authorization` and `Proxy-Authorization` HTTP headers, and censors those strings in the response.

You should set this option to true for transactions that use Basic Authentication because they may contain the base64 unencrypted username and password.


### `send_headers` [_send_headers]

A list of header names to capture and send to Elasticsearch. These headers are placed under the `headers` dictionary in the resulting JSON.


### `send_all_headers` [_send_all_headers]

Instead of sending a white list of headers to Elasticsearch, you can send all headers by setting this option to true. The default is false.


### `redact_headers` [_redact_headers]

A list of headers to redact if present in the HTTP request. This will keep the header field present, but will redact it’s value to show the header’s presence.


### `include_body_for` [_include_body_for]

The list of content types for which Packetbeat exports the full HTTP payload. The HTTP body is available under `http.request.body.content` and `http.response.body.content` for these Content-Types.

In addition, if [`send_response`](/reference/packetbeat/common-protocol-options.md#send-response-option) option is enabled, then the HTTP body is exported together with the HTTP headers under `response` and if [`send_request`](/reference/packetbeat/common-protocol-options.md#send-request-option) enabled, then `request` contains the entire HTTP message including the body.

In the following example, the HTML attachments of the HTTP responses are exported under the `response` field and under `http.request.body.content` or `http.response.body.content`:

```yaml
packetbeat.protocols:
- type: http
  ports: [80, 8080]
  send_response: true
  include_body_for: ["text/html"]
```


### `decode_body` [_decode_body]

A boolean flag that controls decoding of HTTP payload. It interprets the `Content-Encoding` and `Transfer-Encoding` headers and uncompresses the entity body. Supported encodings are `gzip` and `deflate`. This option is only applicable in the cases where the HTTP payload is exported, that is, when one of the `include_*_body_for` options is specified or a POST request contains url-encoded parameters.


### `split_cookie` [_split_cookie]

If the `Cookie` or `Set-Cookie` headers are sent, this option controls whether they are split into individual values. For example, with this option set, an HTTP response might result in the following JSON:

```json
"response": {
  "code": 200,
  "headers": {
    "connection": "close",
    "content-language": "en",
    "content-type": "text/html; charset=utf-8",
    "date": "Fri, 21 Nov 2014 17:07:34 GMT",
    "server": "gunicorn/19.1.1",
    "set-cookie": { <1>
      "csrftoken": "S9ZuJF8mvIMT5CL4T1Xqn32wkA6ZSeyf",
      "expires": "Fri, 20-Nov-2015 17:07:34 GMT",
      "max-age": "31449600",
      "path": "/"
    },
    "vary": "Cookie, Accept-Language"
  },
  "status_phrase": "OK"
}
```

1. Note that `set-cookie` is a map containing the cookie names as keys.


The default is false.


### `real_ip_header` [_real_ip_header]

The header field to extract the real IP from. This setting is useful when you want to capture traffic behind a reverse proxy, but you want to get the geo-location information. If this header is present and contains a valid IP addresses, the information is used for the `network.forwarded_ip` field.


### `max_message_size` [_max_message_size]

If an individual HTTP message is larger than this setting (in bytes), it will be trimmed to this size. Unless this value is very small (<1.5K), Packetbeat is able to still correctly follow the transaction and create an event for it. The default is 10485760 (10 MB).



