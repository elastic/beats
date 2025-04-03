---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-http.html
---

# HTTP fields [exported-fields-http]

HTTP-specific event fields.


## http [_http_2]

Information about the HTTP request and response.


## request [_request_2]

HTTP request

**`http.request.headers`**
:   A map containing the captured header fields from the request. Which headers to capture is configurable. If headers with the same header name are present in the message, they will be separated by commas.

type: object


**`http.request.params`**
:   type: alias

alias to: url.query



## response [_response_2]

HTTP response

**`http.response.status_phrase`**
:   The HTTP status phrase.

example: Not Found


**`http.response.headers`**
:   A map containing the captured header fields from the response. Which headers to capture is configurable. If headers with the same header name are present in the message, they will be separated by commas.

type: object


**`http.response.code`**
:   type: alias

alias to: http.response.status_code


**`http.response.phrase`**
:   type: alias

alias to: http.response.status_phrase


