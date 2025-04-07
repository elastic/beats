---
navigation_title: "Winlogbeat"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/configuration-winlogbeat-options.html
---

# Configure Winlogbeat [configuration-winlogbeat-options]


The `winlogbeat` section of the `winlogbeat.yml` config file specifies all options that are specific to Winlogbeat. Most importantly, it contains the list of event logs to monitor.

Here is a sample configuration:

```yaml
winlogbeat.event_logs:
  - name: Application
    ignore_older: 72h
  - name: Security
  - name: System
```


## Configuration options [_configuration_options]

You can specify the following options in the `winlogbeat` section of the `winlogbeat.yml` config file:


### `registry_file` [_registry_file]

The name of the file where Winlogbeat stores information that it uses to resume monitoring after a restart. By default the file is stored as `.winlogbeat.yml` in the directory where the Beat was started. When you run the process as a Windows service, it’s recommended that you set the value to `C:/ProgramData/winlogbeat/.winlogbeat.yml`.

```yaml
winlogbeat.registry_file: C:/ProgramData/winlogbeat/.winlogbeat.yml
```

::::{note}
The forward slashes (/) in the path are automatically changed to backslashes (\) for Windows compatibility. You can use either forward or backslashes. Forward slashes are easier to work with in YAML because there is no need to escape them.
::::



### `registry_flush` [_registry_flush]

The timeout value that controls when registry entries are written to disk (flushed). When an unwritten update exceeds this value, it triggers a write to disk. When flush is set to 0s, the registry is written to disk after each batch of events has been published successfully.

The default value is 5s.

Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`.

```yaml
winlogbeat.registry_flush: 5s
```


### `shutdown_timeout` [_shutdown_timeout]

The amount of time to wait for all events to be published when shutting down. By default there is no shutdown timeout so Winlogbeat will stop without waiting. When you restart it will resume from the last successfully published event in each event log.

In some use cases you do want to wait for the publishing queue to drain before exiting and that’s when you would use this option.

Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`.

```yaml
winlogbeat.shutdown_timeout: 30s
```


### `event_logs` [_event_logs]

A list of entries (called *dictionaries* in YAML) that specify which event logs to monitor. Each entry in the list defines an event log to monitor as well as any information to be associated with the event log (filter, tags, and so on).

```yaml
winlogbeat.event_logs:
  - name: Application
```


### `event_logs.batch_read_size` [_event_logs_batch_read_size]

The maximum number of event log records to read from the Windows API in a single batch. The default batch size is 512. Most Windows versions return an error if the value is larger than 1024. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

Winlogbeat starts a goroutine (a lightweight thread) to read from each individual event log. The goroutine reads a batch of event log records using the Windows API, applies any processors to the events, publishes them to the configured outputs, and waits for an acknowledgement from the outputs before reading additional event log records.


### `event_logs.name` [configuration-winlogbeat-options-event_logs-name]

The name of the event log to monitor. Each dictionary under `event_logs` must have a `name` field, except for those which use a custom XML query. A channel is a named stream of events that transports events from an event source to an event log. Most channels are tied to specific event publishers. You can get a list of available event logs by using the PowerShell [`Get-WinEvent`](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.diagnostics/get-winevent) cmdlet on Windows Vista or newer. Here is a sample of the output from the command:

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
winlogbeat.event_logs:
  - name: Microsoft-Windows-Windows Firewall With Advanced Security/Firewall
```

To read events from an archived `.evtx` file you can specify the `name` as the absolute path (it cannot be relative) to the file. There’s a complete example of how to read from an .evtx file in the [FAQ](/reference/winlogbeat/reading-from-evtx.md).

```yaml
winlogbeat.event_logs:
  - name: 'C:\backup\sysmon-2019.08.evtx'
```

The name key must not be used with custom XML queries.


### `event_logs.id` [_event_logs_id]

A unique identifier for the event log. This key is required when using a custom XML query.

It is used to uniquely identify the event log reader in the registry file. This is useful if multiple event logs are being set up to watch the same channel or file. If an ID is not given, the `event_logs.name` value will be used.

This value must be unique.

```yaml
winlogbeat.event_logs:
  - name: Application
    id: application-logs
    ignore_older: 168h
```


### `event_logs.ignore_older` [_event_logs_ignore_older]

If this option is specified, Winlogbeat filters events that are older than the specified amount of time. Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". This option is useful when you are beginning to monitor an event log that contains older records that you would like to ignore. This field is optional.

```yaml
winlogbeat.event_logs:
  - name: Application
    ignore_older: 168h
```


### `event_logs.forwarded` [_event_logs_forwarded]

A boolean flag to indicate that the log contains only events collected from remote hosts using the Windows Event Collector. The value defaults to true for the ForwardedEvents log and false for any other log. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

This settings allows Winlogbeat to optimize reads for forwarded events that are already rendered. When the value is true Winlogbeat does not attempt to render the event using message files from the host computer. The Windows Event Collector subscription should be configured to use the "RenderedText" format (this is the default) to ensure that the events are distributed with messages and descriptions.


### `event_logs.event_id` [_event_logs_event_id]

A whitelist and blacklist of event IDs. The value is a comma-separated list. The accepted values are single event IDs to include (e.g. 4624), a range of event IDs to include (e.g. 4700-4800), single event IDs to exclude (e.g. -4735), and a range of event IDs to exclude (e.g. -4701-4710). **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

```yaml
winlogbeat.event_logs:
  - name: Security
    event_id: 4624, 4625, 4700-4800, -4735, -4701-4710
```


### `event_logs.language` [_event_logs_language]

The language ID the events will be rendered in. The language will be forced regardless of the system language. A complete list of language IDs can be found [here](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-lcid/a9eac961-e77d-41a6-90a5-ce1a8b0cdb9c). It defaults to `0`, which indicates to use the system language.

```yaml
winlogbeat.event_logs:
  - name: Security
    event_id: 4624, 4625, 4700-4800, -4735
    language: 0x0409 # en-US
```


### `event_logs.level` [_event_logs_level]

A list of event levels to include. The value is a comma-separated list of levels. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

| Level | Value |
| --- | --- |
| critical, crit | 1 |
| error, err | 2 |
| warning, warn | 3 |
| information, info | 0 or 4 |
| verbose | 5 |

```yaml
winlogbeat.event_logs:
  - name: Security
    level: critical, error, warning
```


### `event_logs.provider` [_event_logs_provider]

A list of providers (source names) to include. The value is a YAML list. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

```yaml
winlogbeat.event_logs:
  - name: Application
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


### `event_logs.xml_query` [_event_logs_xml_query]

Provide a custom XML query. This option is mutually exclusive with the `name`, `event_id`, `ignore_older`, `level`, and `provider` options. These options should be included in the XML query directly. Furthermore, an `id` must be provided. Custom XML queries provide more flexibility and advanced options than the simpler query options in Winlogbeat. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

Here is a configuration which will collect DHCP server events from multiple channels:

```yaml
winlogbeat.event_logs:
  - id: dhcp-server-logs
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


### `event_logs.include_xml` [_event_logs_include_xml]

Boolean option that controls if the raw XML representation of an event is included in the data sent by Winlogbeat. The default is false. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

The XML representation of the event is useful for troubleshooting purposes. The data in the fields reported by Winlogbeat can be compared to the data in the XML to diagnose problems.

Example:

```yaml
winlogbeat.event_logs:
  - name: Microsoft-Windows-Windows Defender/Operational
    include_xml: true
```

* This can have a significant impact on performance that can vary depending on your system specs.


### `event_logs.tags` [_event_logs_tags]

A list of tags that the Beat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
winlogbeat.event_logs:
  - name: CustomLog
    tags: ["web"]
```


### `event_logs.fields` [winlogbeat-configuration-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering event data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
winlogbeat.event_logs:
  - name: CustomLog
    fields:
      customer_id: 51415432
```


### `event_logs.fields_under_root` [_event_logs_fields_under_root]

If this option is set to true, the custom [fields](#winlogbeat-configuration-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Winlogbeat, then the custom fields overwrite the other fields.


### `event_logs.processors` [_event_logs_processors]

A list of processors to apply to the data generated by the event log.

See [Processors](/reference/winlogbeat/filtering-enhancing-data.md) for information about specifying processors in your config.


### `event_logs.index` [_event_logs_index]

If present, this formatted string overrides the index for events from this event log (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"winlogbeat-myindex-2019.12.13"`.


### `event_logs.keep_null` [_event_logs_keep_null]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


### `event_logs.no_more_events` [_event_logs_no_more_events]

The action that the event log reader should take when it receives a signal from Windows that there are no more events to read. It can either `wait` for more events to be written (the default behavior) or it can `stop`. The overall Winlogbeat process will stop when all of the individual event log readers have stopped. **This option is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer).**

Setting `no_more_events` to `stop` is useful when reading from archived event log files where you want to read the whole file then exit. There’s a complete example of how to read from an `.evtx` file in the [FAQ](/reference/winlogbeat/reading-from-evtx.md).


### `overwrite_pipelines` [_overwrite_pipelines]

By default Ingest pipelines are not updated if a pipeline with the same ID already exists. If this option is enabled Winlogbeat overwrites pipelines every time a new Elasticsearch connection is established.

The default value is `false`.

