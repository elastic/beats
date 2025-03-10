---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-iis.html
---

# IIS fields [exported-fields-iis]

Module for parsing IIS log files.


## iis [_iis]

Fields from IIS log files.


## access [_access_2]

Contains fields for IIS access logs.

**`iis.access.sub_status`**
:   The HTTP substatus code.

type: long


**`iis.access.win32_status`**
:   The Windows status code.

type: long


**`iis.access.site_name`**
:   The site name and instance number.

type: keyword


**`iis.access.server_name`**
:   The name of the server on which the log file entry was generated.

type: keyword


**`iis.access.cookie`**
:   The content of the cookie sent or received, if any.

type: keyword


**`iis.access.body_received.bytes`**
:   type: alias

alias to: http.request.body.bytes


**`iis.access.body_sent.bytes`**
:   type: alias

alias to: http.response.body.bytes


**`iis.access.server_ip`**
:   type: alias

alias to: destination.address


**`iis.access.method`**
:   type: alias

alias to: http.request.method


**`iis.access.url`**
:   type: alias

alias to: url.path


**`iis.access.query_string`**
:   type: alias

alias to: url.query


**`iis.access.port`**
:   type: alias

alias to: destination.port


**`iis.access.user_name`**
:   type: alias

alias to: user.name


**`iis.access.remote_ip`**
:   type: alias

alias to: source.address


**`iis.access.referrer`**
:   type: alias

alias to: http.request.referrer


**`iis.access.response_code`**
:   type: alias

alias to: http.response.status_code


**`iis.access.http_version`**
:   type: alias

alias to: http.version


**`iis.access.hostname`**
:   type: alias

alias to: host.hostname


**`iis.access.user_agent.device`**
:   type: alias

alias to: user_agent.device.name


**`iis.access.user_agent.name`**
:   type: alias

alias to: user_agent.name


**`iis.access.user_agent.os`**
:   type: alias

alias to: user_agent.os.full_name


**`iis.access.user_agent.os_name`**
:   type: alias

alias to: user_agent.os.name


**`iis.access.user_agent.original`**
:   type: alias

alias to: user_agent.original


**`iis.access.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`iis.access.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`iis.access.geoip.location`**
:   type: alias

alias to: source.geo.location


**`iis.access.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`iis.access.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`iis.access.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code



## error [_error_3]

Contains fields for IIS error logs.

**`iis.error.reason_phrase`**
:   The HTTP reason phrase.

type: keyword


**`iis.error.queue_name`**
:   The IIS application pool name.

type: keyword


**`iis.error.remote_ip`**
:   type: alias

alias to: source.address


**`iis.error.remote_port`**
:   type: alias

alias to: source.port


**`iis.error.server_ip`**
:   type: alias

alias to: destination.address


**`iis.error.server_port`**
:   type: alias

alias to: destination.port


**`iis.error.http_version`**
:   type: alias

alias to: http.version


**`iis.error.method`**
:   type: alias

alias to: http.request.method


**`iis.error.url`**
:   type: alias

alias to: url.original


**`iis.error.response_code`**
:   type: alias

alias to: http.response.status_code


**`iis.error.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`iis.error.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`iis.error.geoip.location`**
:   type: alias

alias to: source.geo.location


**`iis.error.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`iis.error.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`iis.error.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code


