---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-nginx.html
---

# Nginx fields [exported-fields-nginx]

Module for parsing the Nginx log files.


## nginx [_nginx]

Fields from the Nginx log files.


## access [_access_3]

Contains fields for the Nginx access logs.

**`nginx.access.remote_ip_list`**
:   An array of remote IP addresses. It is a list because it is common to include, besides the client IP address, IP addresses from headers like `X-Forwarded-For`. Real source IP is restored to `source.ip`.

type: array


**`nginx.access.body_sent.bytes`**
:   type: alias

alias to: http.response.body.bytes


**`nginx.access.user_name`**
:   type: alias

alias to: user.name


**`nginx.access.method`**
:   type: alias

alias to: http.request.method


**`nginx.access.url`**
:   type: alias

alias to: url.original


**`nginx.access.http_version`**
:   type: alias

alias to: http.version


**`nginx.access.response_code`**
:   type: alias

alias to: http.response.status_code


**`nginx.access.referrer`**
:   type: alias

alias to: http.request.referrer


**`nginx.access.agent`**
:   type: alias

alias to: user_agent.original


**`nginx.access.user_agent.device`**
:   type: alias

alias to: user_agent.device.name


**`nginx.access.user_agent.name`**
:   type: alias

alias to: user_agent.name


**`nginx.access.user_agent.os`**
:   type: alias

alias to: user_agent.os.full_name


**`nginx.access.user_agent.os_name`**
:   type: alias

alias to: user_agent.os.name


**`nginx.access.user_agent.original`**
:   type: alias

alias to: user_agent.original


**`nginx.access.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`nginx.access.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`nginx.access.geoip.location`**
:   type: alias

alias to: source.geo.location


**`nginx.access.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`nginx.access.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`nginx.access.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code



## error [_error_5]

Contains fields for the Nginx error logs.

**`nginx.error.connection_id`**
:   Connection identifier.

type: long


**`nginx.error.level`**
:   type: alias

alias to: log.level


**`nginx.error.pid`**
:   type: alias

alias to: process.pid


**`nginx.error.tid`**
:   type: alias

alias to: process.thread.id


**`nginx.error.message`**
:   type: alias

alias to: message



## ingress_controller [_ingress_controller]

Contains fields for the Ingress Nginx controller access logs.

**`nginx.ingress_controller.remote_ip_list`**
:   An array of remote IP addresses. It is a list because it is common to include, besides the client IP address, IP addresses from headers like `X-Forwarded-For`. Real source IP is restored to `source.ip`.

type: array


**`nginx.ingress_controller.upstream_address_list`**
:   An array of the upstream addresses. It is a list because it is common that several upstream servers were contacted during request processing.

type: keyword


**`nginx.ingress_controller.upstream.response.length_list`**
:   An array of upstream response lengths. It is a list because it is common that several upstream servers were contacted during request processing.

type: keyword


**`nginx.ingress_controller.upstream.response.time_list`**
:   An array of upstream response durations. It is a list because it is common that several upstream servers were contacted during request processing.

type: keyword


**`nginx.ingress_controller.upstream.response.status_code_list`**
:   An array of upstream response status codes. It is a list because it is common that several upstream servers were contacted during request processing.

type: keyword


**`nginx.ingress_controller.http.request.length`**
:   The request length (including request line, header, and request body)

type: long

format: bytes


**`nginx.ingress_controller.http.request.time`**
:   Time elapsed since the first bytes were read from the client

type: double

format: duration


**`nginx.ingress_controller.upstream.name`**
:   The name of the upstream.

type: keyword


**`nginx.ingress_controller.upstream.alternative_name`**
:   The name of the alternative upstream.

type: keyword


**`nginx.ingress_controller.upstream.response.length`**
:   The length of the response obtained from the upstream server. If several servers were contacted during request process, the summary of the multiple response lengths is stored.

type: long

format: bytes


**`nginx.ingress_controller.upstream.response.time`**
:   The time spent on receiving the response from the upstream as seconds with millisecond resolution. If several servers were contacted during request process, the summary of the multiple response times is stored.

type: double

format: duration


**`nginx.ingress_controller.upstream.response.status_code`**
:   The status code of the response obtained from the upstream server. If several servers were contacted during request process, only the status code of the response from the last one is stored in this field.

type: long


**`nginx.ingress_controller.upstream.ip`**
:   The IP address of the upstream server. If several servers were contacted during request process, only the last one is stored in this field.

type: ip


**`nginx.ingress_controller.upstream.port`**
:   The port of the upstream server. If several servers were contacted during request process, only the last one is stored in this field.

type: long


**`nginx.ingress_controller.http.request.id`**
:   The randomly generated ID of the request

type: keyword


**`nginx.ingress_controller.body_sent.bytes`**
:   type: alias

alias to: http.response.body.bytes


**`nginx.ingress_controller.user_name`**
:   type: alias

alias to: user.name


**`nginx.ingress_controller.method`**
:   type: alias

alias to: http.request.method


**`nginx.ingress_controller.url`**
:   type: alias

alias to: url.original


**`nginx.ingress_controller.http_version`**
:   type: alias

alias to: http.version


**`nginx.ingress_controller.response_code`**
:   type: alias

alias to: http.response.status_code


**`nginx.ingress_controller.referrer`**
:   type: alias

alias to: http.request.referrer


**`nginx.ingress_controller.agent`**
:   type: alias

alias to: user_agent.original


**`nginx.ingress_controller.user_agent.device`**
:   type: alias

alias to: user_agent.device.name


**`nginx.ingress_controller.user_agent.name`**
:   type: alias

alias to: user_agent.name


**`nginx.ingress_controller.user_agent.os`**
:   type: alias

alias to: user_agent.os.full_name


**`nginx.ingress_controller.user_agent.os_name`**
:   type: alias

alias to: user_agent.os.name


**`nginx.ingress_controller.user_agent.original`**
:   type: alias

alias to: user_agent.original


**`nginx.ingress_controller.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`nginx.ingress_controller.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`nginx.ingress_controller.geoip.location`**
:   type: alias

alias to: source.geo.location


**`nginx.ingress_controller.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`nginx.ingress_controller.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`nginx.ingress_controller.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code


