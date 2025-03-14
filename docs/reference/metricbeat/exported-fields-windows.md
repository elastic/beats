---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-windows.html
---

# Windows fields [exported-fields-windows]

Module for Windows


## windows [_windows]


## perfmon [_perfmon]

perfmon

**`windows.perfmon.instance`**
:   Instance value.

type: keyword


**`windows.perfmon.metrics.*.*`**
:   Metric values returned.

type: object



## service [_service_5]

`service` contains the status for Windows services.

**`windows.service.id`**
:   A unique ID for the service. It is a hash of the machine’s GUID and the service name.

type: keyword

example: hW3NJFc1Ap


**`windows.service.name`**
:   The service name.

type: keyword

example: Wecsvc


**`windows.service.display_name`**
:   The display name of the service.

type: keyword

example: Windows Event Collector


**`windows.service.start_type`**
:   The startup type of the service. The possible values are `Automatic`, `Boot`, `Disabled`, `Manual`, and `System`.

type: keyword


**`windows.service.start_name`**
:   Account name under which a service runs.

type: keyword

example: NT AUTHORITY\LocalService


**`windows.service.path_name`**
:   Fully qualified path to the file that implements the service, including arguments.

type: keyword

example: C:\WINDOWS\system32\svchost.exe -k LocalService -p


**`windows.service.state`**
:   The actual state of the service. The possible values are `Continuing`, `Pausing`, `Paused`, `Running`, `Starting`, `Stopping`, and `Stopped`.

type: keyword


**`windows.service.exit_code`**
:   For `Stopped` services this is the error code that service reports when starting to stopping. This will be the generic Windows service error code unless the service provides a service-specific error code.

type: keyword


**`windows.service.pid`**
:   For `Running` services this is the associated process PID.

type: long

example: 1092


**`windows.service.uptime.ms`**
:   The service’s uptime specified in milliseconds.

type: long

format: duration



## wmi [_wmi]

wmi

