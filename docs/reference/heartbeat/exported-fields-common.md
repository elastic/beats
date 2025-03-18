---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-common.html
---

# Common heartbeat monitor fields [exported-fields-common]

None


## monitor [_monitor]

Common monitor fields.

**`monitor.type`**
:   The monitor type.

type: keyword


**`monitor.name`**
:   The monitors configured name

type: keyword


**`monitor.name.text`**
:   type: text


**`monitor.id`**
:   The monitors full job ID as used by heartbeat.

type: keyword


**`monitor.id.text`**
:   type: text



## duration [_duration_2]

Total monitoring test duration

**`monitor.duration.us`**
:   Duration in microseconds

type: long


**`monitor.scheme`**
:   Address url scheme. For example `tcp`, `tls`, `http`, and `https`.

type: alias

alias to: url.scheme


**`monitor.host`**
:   Hostname of service being monitored. Can be missing, if service is monitored by IP.

type: alias

alias to: url.domain


**`monitor.ip`**
:   IP of service being monitored. If service is monitored by hostname, the `ip` field contains the resolved ip address for the current host.

type: ip


**`monitor.status`**
:   Indicator if monitor could validate the service to be available.

type: keyword

required: True


**`monitor.check_group`**
:   A token unique to a simultaneously invoked group of checks as in the case where multiple IPs are checked for a single DNS entry.

type: keyword


**`monitor.timespan`**
:   Time range this ping reported starting at the instant the check was started, ending at the start of the next scheduled check.

type: date_range


**`monitor.origin`**
:   The origin of this monitor configuration, usually either "ui", or "project"

type: keyword



## project [_project]

Project info for this monitor

**`monitor.project.id`**
:   Project ID

type: keyword


**`monitor.project.name`**
:   Project name

type: keyword


