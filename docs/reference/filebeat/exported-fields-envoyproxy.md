---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-envoyproxy.html
---

# Envoyproxy fields [exported-fields-envoyproxy]

Module for handling logs produced by envoy


## envoyproxy [_envoyproxy]

Fields from envoy proxy logs after normalization

**`envoyproxy.log_type`**
:   Envoy log type, normally ACCESS

type: keyword


**`envoyproxy.response_flags`**
:   Response flags

type: keyword


**`envoyproxy.upstream_service_time`**
:   Upstream service time in nanoseconds

type: long

format: duration


**`envoyproxy.request_id`**
:   ID of the request

type: keyword


**`envoyproxy.authority`**
:   Envoy proxy authority field

type: keyword


**`envoyproxy.proxy_type`**
:   Envoy proxy type, tcp or http

type: keyword


