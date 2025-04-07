---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-haproxy.html
---

# HAProxy fields [exported-fields-haproxy]

haproxy Module


## haproxy [_haproxy]

**`haproxy.frontend_name`**
:   Name of the frontend (or listener) which received and processed the connection.


**`haproxy.backend_name`**
:   Name of the backend (or listener) which was selected to manage the connection to the server.


**`haproxy.server_name`**
:   Name of the last server to which the connection was sent.


**`haproxy.total_waiting_time_ms`**
:   Total time in milliseconds spent waiting in the various queues

type: long


**`haproxy.connection_wait_time_ms`**
:   Total time in milliseconds spent waiting for the connection to establish to the final server

type: long


**`haproxy.bytes_read`**
:   Total number of bytes transmitted to the client when the log is emitted.

type: long


**`haproxy.time_queue`**
:   Total time in milliseconds spent waiting in the various queues.

type: long


**`haproxy.time_backend_connect`**
:   Total time in milliseconds spent waiting for the connection to establish to the final server, including retries.

type: long


**`haproxy.server_queue`**
:   Total number of requests which were processed before this one in the server queue.

type: long


**`haproxy.backend_queue`**
:   Total number of requests which were processed before this one in the backend’s global queue.

type: long


**`haproxy.bind_name`**
:   Name of the listening address which received the connection.


**`haproxy.error_message`**
:   Error message logged by HAProxy in case of error.

type: text


**`haproxy.source`**
:   The HAProxy source of the log

type: keyword


**`haproxy.termination_state`**
:   Condition the session was in when the session ended.


**`haproxy.mode`**
:   mode that the frontend is operating (TCP or HTTP)

type: keyword



## connections [_connections]

Contains various counts of connections active in the process.

**`haproxy.connections.active`**
:   Total number of concurrent connections on the process when the session was logged.

type: long


**`haproxy.connections.frontend`**
:   Total number of concurrent connections on the frontend when the session was logged.

type: long


**`haproxy.connections.backend`**
:   Total number of concurrent connections handled by the backend when the session was logged.

type: long


**`haproxy.connections.server`**
:   Total number of concurrent connections still active on the server when the session was logged.

type: long


**`haproxy.connections.retries`**
:   Number of connection retries experienced by this session when trying to connect to the server.

type: long



## client [_client_2]

Information about the client doing the request

**`haproxy.client.ip`**
:   type: alias

alias to: source.address


**`haproxy.client.port`**
:   type: alias

alias to: source.port


**`haproxy.process_name`**
:   type: alias

alias to: process.name


**`haproxy.pid`**
:   type: alias

alias to: process.pid



## destination [_destination_2]

Destination information

**`haproxy.destination.port`**
:   type: alias

alias to: destination.port


**`haproxy.destination.ip`**
:   type: alias

alias to: destination.ip



## geoip [_geoip]

Contains GeoIP information gathered based on the client.ip field. Only present if the GeoIP Elasticsearch plugin is available and used.

**`haproxy.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`haproxy.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`haproxy.geoip.location`**
:   type: alias

alias to: source.geo.location


**`haproxy.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`haproxy.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`haproxy.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code



## http [_http_2]

Please add description


## response [_response_2]

Fields related to the HTTP response

**`haproxy.http.response.captured_cookie`**
:   Optional "name=value" entry indicating that the client had this cookie in the response.


**`haproxy.http.response.captured_headers`**
:   List of headers captured in the response due to the presence of the "capture response header" statement in the frontend.

type: keyword


**`haproxy.http.response.status_code`**
:   type: alias

alias to: http.response.status_code



## request [_request_2]

Fields related to the HTTP request

**`haproxy.http.request.captured_cookie`**
:   Optional "name=value" entry indicating that the server has returned a cookie with its request.


**`haproxy.http.request.captured_headers`**
:   List of headers captured in the request due to the presence of the "capture request header" statement in the frontend.

type: keyword


**`haproxy.http.request.raw_request_line`**
:   Complete HTTP request line, including the method, request and HTTP version string.

type: keyword


**`haproxy.http.request.time_wait_without_data_ms`**
:   Total time in milliseconds spent waiting for the server to send a full HTTP response, not counting data.

type: long


**`haproxy.http.request.time_wait_ms`**
:   Total time in milliseconds spent waiting for a full HTTP request from the client (not counting body) after the first byte was received.

type: long



## tcp [_tcp]

TCP log format

**`haproxy.tcp.connection_waiting_time_ms`**
:   Total time in milliseconds elapsed between the accept and the last close

type: long


