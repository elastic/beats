---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-traefik.html
---

# Traefik fields [exported-fields-traefik]

Module for parsing the Traefik log files.


## traefik [_traefik]

Fields from the Traefik log files.


## access [_access_4]

Contains fields for the Traefik access logs.

**`traefik.access.user_identifier`**
:   Is the RFC 1413 identity of the client

type: keyword


**`traefik.access.request_count`**
:   The number of requests

type: long


**`traefik.access.frontend_name`**
:   The name of the frontend used

type: keyword


**`traefik.access.backend_url`**
:   The url of the backend where request is forwarded

type: keyword


**`traefik.access.body_sent.bytes`**
:   type: alias

alias to: http.response.body.bytes


**`traefik.access.remote_ip`**
:   type: alias

alias to: source.address


**`traefik.access.user_name`**
:   type: alias

alias to: user.name


**`traefik.access.method`**
:   type: alias

alias to: http.request.method


**`traefik.access.url`**
:   type: alias

alias to: url.original


**`traefik.access.http_version`**
:   type: alias

alias to: http.version


**`traefik.access.response_code`**
:   type: alias

alias to: http.response.status_code


**`traefik.access.referrer`**
:   type: alias

alias to: http.request.referrer


**`traefik.access.agent`**
:   type: alias

alias to: user_agent.original


**`traefik.access.user_agent.name`**
:   type: alias

alias to: user_agent.name


**`traefik.access.user_agent.os`**
:   type: alias

alias to: user_agent.os.full_name


**`traefik.access.user_agent.os_name`**
:   type: alias

alias to: user_agent.os.name


**`traefik.access.user_agent.original`**
:   type: alias

alias to: user_agent.original


**`traefik.access.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`traefik.access.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`traefik.access.geoip.location`**
:   type: alias

alias to: source.geo.location


**`traefik.access.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`traefik.access.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`traefik.access.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code


