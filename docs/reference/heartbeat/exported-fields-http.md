---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-http.html
---

# HTTP monitor fields [exported-fields-http]

None


## http [_http_2]

HTTP related fields.

**`http.url`**
:   Service url used by monitor.

type: alias

alias to: url.full


**`http.response.body.hash`**
:   Hash of the full response body. Can be used to group responses with identical hashes.

type: keyword


**`http.response.redirects`**
:   List of redirects followed to arrive at final content. Last item on the list is the URL for which body content is shown.

type: keyword


**`http.response.headers.*`**
:   The canonical headers of the monitored HTTP response.

type: object

Object is not enabled.



## rtt [_rtt]

HTTP layer round trip times.


## validate [_validate]

Duration between first byte of HTTP request being written and response being processed by validator. Duration based on already available network connection.

Note: if validator is not reading body or only a prefix, this number does not fully represent the total time needed to read the body.

**`http.rtt.validate.us`**
:   Duration in microseconds

type: long



## validate_body [_validate_body]

Duration of validator required to read and validate the response body.

Note: if validator is not reading body or only a prefix, this number does not fully represent the total time needed to read the body.

**`http.rtt.validate_body.us`**
:   Duration in microseconds

type: long



## write_request [_write_request]

Duration of sending the complete HTTP request. Duration based on already available network connection.

**`http.rtt.write_request.us`**
:   Duration in microseconds

type: long



## response_header [_response_header]

Time required between sending the start of sending the HTTP request and first byte from HTTP response being read. Duration based on already available network connection.

**`http.rtt.response_header.us`**
:   Duration in microseconds

type: long


**`http.rtt.content.us`**
:   Time required to retrieved the content in micro seconds.

type: long



## total [_total]

Duration required to process the HTTP transaction. Starts with the initial TCP connection attempt. Ends with after validator did check the response.

Note: if validator is not reading body or only a prefix, this number does not fully represent the total time needed.

**`http.rtt.total.us`**
:   Duration in microseconds

type: long


