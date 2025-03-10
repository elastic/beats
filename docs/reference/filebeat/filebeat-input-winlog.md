---
navigation_title: "winlog"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-winlog.html
---

# winlog input [filebeat-input-winlog]


::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Use the `winlog` input to read Windows event logs. It reads from one event log using Windows APIs, filters the events based on user-configured criteria, then sends the event data to the configured outputs. It watches the event log so that new event data is sent in a timely manner. The read position for the event log is persisted to disk to allow the input to resume after restarts.

The `winlog` input can capture event data from any event logs running on your system. For example, you can capture events such as:

* application events
* hardware events
* security events
* system events

Here is a sample configuration:

```yaml
- type: winlog
  name: Application
  ignore_older: 72h
```


## Configuration options [_configuration_options_23]


### `batch_read_size` [_batch_read_size]

The maximum number of event log records to read from the Windows API in a single batch. The default batch size is 512. Most Windows versions return an error if the value is larger than 1024. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

Filebeat starts a goroutine (a lightweight thread) to read from each individual event log. The goroutine reads a batch of event log records using the Windows API, applies any processors to the events, publishes them to the configured outputs, and waits for an acknowledgement from the outputs before reading additional event log records.


### `name` [_name]

The name of the event log to monitor. It must have a `name` field, except for those which use a custom XML query. A channel is a named stream of events that transports events from an event source to an event log. Most channels are tied to specific event publishers. You can get a list of available event logs by using the PowerShell [`Get-WinEvent`](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.diagnostics/get-winevent) cmdlet on Windows Vista or newer. Here is a sample of the output from the command:

```sh
PS C:\> Get-WinEvent -ListLog * | Format-List -Property LogName
LogName : Application
LogName : HardwareEvents
LogName : Internet Explorer
LogName : Key Management Service
LogName : Security
LogName : System
LogName : Windows PowerShell
LogName : ForwardedEvents
LogName : Microsoft-Management-UI/Admin
LogName : Microsoft-Rdms-UI/Admin
LogName : Microsoft-Rdms-UI/Operational
LogName : Microsoft-Windows-Windows Firewall With Advanced Security/Firewall
...
```

If `Get-WinEvent` is not available, the [`Get-EventLog`](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.management/get-eventlog) cmdlet can be used in its place.

```sh
PS C:\Users\vagrant> Get-EventLog *

  Max(K) Retain OverflowAction        Entries Log
  ------ ------ --------------        ------- ---
  20,480      0 OverwriteAsNeeded          75 Application
  20,480      0 OverwriteAsNeeded           0 HardwareEvents
     512      7 OverwriteOlder              0 Internet Explorer
  20,480      0 OverwriteAsNeeded           0 Key Management Service
  20,480      0 OverwriteAsNeeded       1,609 Security
  20,480      0 OverwriteAsNeeded       1,184 System
  15,360      0 OverwriteAsNeeded         464 Windows PowerShell
```

You must specify the full name of the channel in the configuration file.

```yaml
- type: winlog
  name: Microsoft-Windows-Windows Firewall With Advanced Security/Firewall
```

To read events from an archived `.evtx` file you can specify the `name` as the absolute path (it cannot be relative) to the file.

```yaml
- type: winlog
  name: 'C:\backup\sysmon-2019.08.evtx'
  no_more_events: stop
```

The name key must not be used with custom XML queries.


### `id` [_id]

A unique identifier for the event log. This key is required when using a custom XML query.

It is used to uniquely identify the event log reader in the registry file. This is useful if multiple event logs are being set up to watch the same channel or file. If an ID is not given, the `name` value will be used.

This value must be unique.

```yaml
- type: winlog
  name: Application
  id: application-logs
  ignore_older: 168h
```


### `ignore_older` [_ignore_older_2]

If this option is specified, the input filters events that are older than the specified amount of time. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". This option is useful when you are beginning to monitor an event log that contains older records that you would like to ignore. This field is optional.

```yaml
- type: winlog
  name: Application
  ignore_older: 168h
```


### `forwarded` [_forwarded]

A boolean flag to indicate that the log contains only events collected from remote hosts using the Windows Event Collector. The value defaults to true for the ForwardedEvents log and false for any other log. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

This settings allows Filebeat to optimize reads for forwarded events that are already rendered. When the value is true Filebeat does not attempt to render the event using message files from the host computer. The Windows Event Collector subscription should be configured to use the "RenderedText" format (this is the default) to ensure that the events are distributed with messages and descriptions.


### `event_id` [_event_id]

An allowlist and blocklist of event IDs. The value is a comma-separated list. The accepted values are single event IDs to include (e.g. 4624), a range of event IDs to include (e.g. 4700-4800), and single event IDs to exclude (e.g. -4735). **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

```yaml
- type: winlog
  name: Security
  event_id: 4624, 4625, 4700-4800, -4735
```


### `language` [_language]

The language ID the events will be rendered in. The language will be forced regardless of the system language. A complete list of language IDs can be found [here](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-lcid/a9eac961-e77d-41a6-90a5-ce1a8b0cdb9c). It defaults to `0`, which indicates to use the system language.

```yaml
- type: winlog
  name: Security
  event_id: 4624, 4625, 4700-4800, -4735
  language: 0x0409 # en-US
```


### `level` [_level]

A list of event levels to include. The value is a comma-separated list of levels. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

| Level | Value |
| --- | --- |
| critical, crit | 1 |
| error, err | 2 |
| warning, warn | 3 |
| information, info | 0 or 4 |
| verbose | 5 |

```yaml
- type: winlog
  name: Security
  level: critical, error, warning
```


### `provider` [_provider_3]

A list of providers (source names) to include. The value is a YAML list. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

```yaml
- type: winlog
  name: Application
  provider:
    - Application Error
    - Application Hang
    - Windows Error Reporting
    - EMET
```

You can obtain a list of providers associated with a log by using PowerShell. Here is an example showing the providers associated with the Security log.

```sh
PS C:\> (Get-WinEvent -ListLog Security).ProviderNames
DS
LSA
SC Manager
Security
Security Account Manager
ServiceModel 4.0.0.0
Spooler
TCP/IP
VSSAudit
Microsoft-Windows-Security-Auditing
Microsoft-Windows-Eventlog
```


### `xml_query` [_xml_query]

Provide a custom XML query. This option is mutually exclusive with the `name`, `event_id`, `ignore_older`, `level`, and `provider` options. These options should be included in the XML query directly. Furthermore, an `id` must be provided. Custom XML queries provide more flexibility and advanced options than the simpler query options in Filebeat. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

Here is a configuration which will collect DHCP server events from multiple channels:

```yaml
- type: winlog
  id: dhcp-server-logs
  xml_query: >
    <QueryList>
      <Query Id="0" Path="DhcpAdminEvents">
        <Select Path="DhcpAdminEvents">*</Select>
        <Select Path="Microsoft-Windows-Dhcp-Server/FilterNotifications">*</Select>
        <Select Path="Microsoft-Windows-Dhcp-Server/Operational">*</Select>
      </Query>
    </QueryList>
```

XML queries may also be created in Windows Event Viewer using custom views. The query can be created using a graphical interface and the corresponding XML can be retrieved from the XML tab.


### `include_xml` [_include_xml]

Boolean option that controls if the raw XML representation of an event is included in the data sent by Filebeat. The default is false. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

The XML representation of the event is useful for troubleshooting purposes. The data in the fields reported by Filebeat can be compared to the data in the XML to diagnose problems.

Example:

```yaml
- type: winlog
  name: Microsoft-Windows-Windows Defender/Operational
  include_xml: true
```

* This can have a significant impact on performance that can vary depending on your system specs.


### `tags` [_tags_29]

A list of tags that the Beat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
- type: winlog
  name: CustomLog
  tags: ["web"]
```


### `fields` [winlog-configuration-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering event data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
- type: winlog
  name: CustomLog
  fields:
    customer_id: 51415432
```


### `fields_under_root` [_fields_under_root]

If this option is set to true, the custom [fields](#winlog-configuration-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


### `processors` [_processors_29]

A list of processors to apply to the data generated by the event log.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


### `index` [_index_29]

If present, this formatted string overrides the index for events from this event log (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.12.13"`.


### `keep_null` [_keep_null_29]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


### `no_more_events` [_no_more_events]

The action that the event log reader should take when it receives a signal from Windows that there are no more events to read. It can either `wait` for more events to be written (the default behavior) or it can `stop`. The overall Filebeat process will stop when all of the individual event log readers have stopped. **{This option is only available on operating systems +
  supporting the Windows Event Log API (Microsoft Windows Vista and newer).}**

Setting `no_more_events` to `stop` is useful when reading from archived event log files where you want to read the whole file then exit.

