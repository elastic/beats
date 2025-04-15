---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/exported-fields-winlog.html
---

# Winlogbeat fields [exported-fields-winlog]

Fields from the Windows Event Log.

**`event.original`**
:   The raw XML representation of the event obtained from Windows. This field is only available on operating systems supporting the Windows Event Log API (Microsoft Windows Vista and newer). This field is not included by default and must be enabled by setting `include_xml: true` as a configuration option for an individual event log. The XML representation of the event is useful for troubleshooting purposes. The data in the fields reported by Winlogbeat can be compared to the data in the XML to diagnose problems.



## winlog [_winlog]

All fields specific to the Windows Event Log are defined here.

**`winlog.activity_id`**
:   A globally unique identifier that identifies the current activity. The events that are published with this identifier are part of the same activity.

type: keyword

required: False


**`winlog.computer_name`**
:   The name of the computer that generated the record. When using Windows event forwarding, this name can differ from `agent.hostname`.

type: keyword

required: True


**`winlog.computerObject.domain`**
:   The domain of the account that was added, modified or deleted in the event.

type: keyword

required: False


**`winlog.computerObject.id`**
:   A globally unique identifier that identifies the target device.

type: keyword

required: False


**`winlog.computerObject.name`**
:   The account name that was added, modified or deleted in the event.

type: keyword

required: False


**`winlog.event_data`**
:   The event-specific data. This field is mutually exclusive with `user_data`. If you are capturing event data on versions prior to Windows Vista, the parameters in `event_data` are named `param1`, `param2`, and so on, because event log parameters are unnamed in earlier versions of Windows.

type: object

required: False



## event_data [_event_data]

This is a non-exhaustive list of parameters that are used in Windows events. By having these fields defined in the template they can be used in dashboards and machine-learning jobs.

**`winlog.event_data.AuthenticationPackageName`**
:   type: keyword


**`winlog.event_data.Binary`**
:   type: keyword


**`winlog.event_data.BitlockerUserInputTime`**
:   type: keyword


**`winlog.event_data.BootMode`**
:   type: keyword


**`winlog.event_data.BootType`**
:   type: keyword


**`winlog.event_data.BuildVersion`**
:   type: keyword


**`winlog.event_data.CallTrace`**
:   type: keyword


**`winlog.event_data.ClientInfo`**
:   type: keyword


**`winlog.event_data.Company`**
:   type: keyword


**`winlog.event_data.Configuration`**
:   type: keyword


**`winlog.event_data.CorruptionActionState`**
:   type: keyword


**`winlog.event_data.CreationUtcTime`**
:   type: keyword


**`winlog.event_data.Description`**
:   type: keyword


**`winlog.event_data.Detail`**
:   type: keyword


**`winlog.event_data.DeviceName`**
:   type: keyword


**`winlog.event_data.DeviceNameLength`**
:   type: keyword


**`winlog.event_data.DeviceTime`**
:   type: keyword


**`winlog.event_data.DeviceVersionMajor`**
:   type: keyword


**`winlog.event_data.DeviceVersionMinor`**
:   type: keyword


**`winlog.event_data.DriveName`**
:   type: keyword


**`winlog.event_data.DriverName`**
:   type: keyword


**`winlog.event_data.DriverNameLength`**
:   type: keyword


**`winlog.event_data.DwordVal`**
:   type: keyword


**`winlog.event_data.EntryCount`**
:   type: keyword


**`winlog.event_data.EventType`**
:   type: keyword


**`winlog.event_data.EventNamespace`**
:   type: keyword


**`winlog.event_data.ExtraInfo`**
:   type: keyword


**`winlog.event_data.FailureName`**
:   type: keyword


**`winlog.event_data.FailureNameLength`**
:   type: keyword


**`winlog.event_data.FileVersion`**
:   type: keyword


**`winlog.event_data.FinalStatus`**
:   type: keyword


**`winlog.event_data.GrantedAccess`**
:   type: keyword


**`winlog.event_data.Group`**
:   type: keyword


**`winlog.event_data.IdleImplementation`**
:   type: keyword


**`winlog.event_data.IdleStateCount`**
:   type: keyword


**`winlog.event_data.ImpersonationLevel`**
:   type: keyword


**`winlog.event_data.IntegrityLevel`**
:   type: keyword


**`winlog.event_data.IpAddress`**
:   type: keyword


**`winlog.event_data.IpPort`**
:   type: keyword


**`winlog.event_data.KeyLength`**
:   type: keyword


**`winlog.event_data.LastBootGood`**
:   type: keyword


**`winlog.event_data.LastShutdownGood`**
:   type: keyword


**`winlog.event_data.LmPackageName`**
:   type: keyword


**`winlog.event_data.LogonGuid`**
:   type: keyword


**`winlog.event_data.LogonId`**
:   type: keyword


**`winlog.event_data.LogonProcessName`**
:   type: keyword


**`winlog.event_data.LogonType`**
:   type: keyword


**`winlog.event_data.MajorVersion`**
:   type: keyword


**`winlog.event_data.MaximumPerformancePercent`**
:   type: keyword


**`winlog.event_data.MemberName`**
:   type: keyword


**`winlog.event_data.MemberSid`**
:   type: keyword


**`winlog.event_data.MinimumPerformancePercent`**
:   type: keyword


**`winlog.event_data.MinimumThrottlePercent`**
:   type: keyword


**`winlog.event_data.MinorVersion`**
:   type: keyword


**`winlog.event_data.Name`**
:   type: keyword


**`winlog.event_data.NewProcessId`**
:   type: keyword


**`winlog.event_data.NewProcessName`**
:   type: keyword


**`winlog.event_data.NewSchemeGuid`**
:   type: keyword


**`winlog.event_data.NewThreadId`**
:   type: keyword


**`winlog.event_data.NewTime`**
:   type: keyword


**`winlog.event_data.NominalFrequency`**
:   type: keyword


**`winlog.event_data.Number`**
:   type: keyword


**`winlog.event_data.OldSchemeGuid`**
:   type: keyword


**`winlog.event_data.OldTime`**
:   type: keyword


**`winlog.event_data.Operation`**
:   type: keyword


**`winlog.event_data.OriginalFileName`**
:   type: keyword


**`winlog.event_data.Path`**
:   type: keyword


**`winlog.event_data.PerformanceImplementation`**
:   type: keyword


**`winlog.event_data.PreviousCreationUtcTime`**
:   type: keyword


**`winlog.event_data.PreviousTime`**
:   type: keyword


**`winlog.event_data.PrivilegeList`**
:   type: keyword


**`winlog.event_data.ProcessId`**
:   type: keyword


**`winlog.event_data.ProcessName`**
:   type: keyword


**`winlog.event_data.ProcessPath`**
:   type: keyword


**`winlog.event_data.ProcessPid`**
:   type: keyword


**`winlog.event_data.Product`**
:   type: keyword


**`winlog.event_data.PuaCount`**
:   type: keyword


**`winlog.event_data.PuaPolicyId`**
:   type: keyword


**`winlog.event_data.QfeVersion`**
:   type: keyword


**`winlog.event_data.Query`**
:   type: keyword


**`winlog.event_data.Reason`**
:   type: keyword


**`winlog.event_data.SchemaVersion`**
:   type: keyword


**`winlog.event_data.ScriptBlockText`**
:   type: keyword


**`winlog.event_data.ServiceName`**
:   type: keyword


**`winlog.event_data.ServiceVersion`**
:   type: keyword


**`winlog.event_data.Session`**
:   type: keyword


**`winlog.event_data.ShutdownActionType`**
:   type: keyword


**`winlog.event_data.ShutdownEventCode`**
:   type: keyword


**`winlog.event_data.ShutdownReason`**
:   type: keyword


**`winlog.event_data.Signature`**
:   type: keyword


**`winlog.event_data.SignatureStatus`**
:   type: keyword


**`winlog.event_data.Signed`**
:   type: keyword


**`winlog.event_data.StartAddress`**
:   type: keyword


**`winlog.event_data.StartFunction`**
:   type: keyword


**`winlog.event_data.StartModule`**
:   type: keyword


**`winlog.event_data.StartTime`**
:   type: keyword


**`winlog.event_data.State`**
:   type: keyword


**`winlog.event_data.Status`**
:   type: keyword


**`winlog.event_data.StopTime`**
:   type: keyword


**`winlog.event_data.SubjectDomainName`**
:   type: keyword


**`winlog.event_data.SubjectLogonId`**
:   type: keyword


**`winlog.event_data.SubjectUserName`**
:   type: keyword


**`winlog.event_data.SubjectUserSid`**
:   type: keyword


**`winlog.event_data.TSId`**
:   type: keyword


**`winlog.event_data.TargetDomainName`**
:   type: keyword


**`winlog.event_data.TargetImage`**
:   type: keyword


**`winlog.event_data.TargetInfo`**
:   type: keyword


**`winlog.event_data.TargetLogonGuid`**
:   type: keyword


**`winlog.event_data.TargetLogonId`**
:   type: keyword


**`winlog.event_data.TargetProcessGUID`**
:   type: keyword


**`winlog.event_data.TargetProcessId`**
:   type: keyword


**`winlog.event_data.TargetServerName`**
:   type: keyword


**`winlog.event_data.TargetUserName`**
:   type: keyword


**`winlog.event_data.TargetUserSid`**
:   type: keyword


**`winlog.event_data.TerminalSessionId`**
:   type: keyword


**`winlog.event_data.TokenElevationType`**
:   type: keyword


**`winlog.event_data.TransmittedServices`**
:   type: keyword


**`winlog.event_data.Type`**
:   type: keyword


**`winlog.event_data.UserSid`**
:   type: keyword


**`winlog.event_data.Version`**
:   type: keyword


**`winlog.event_data.Workstation`**
:   type: keyword


**`winlog.event_data.param1`**
:   type: keyword


**`winlog.event_data.param2`**
:   type: keyword


**`winlog.event_data.param3`**
:   type: keyword


**`winlog.event_data.param4`**
:   type: keyword


**`winlog.event_data.param5`**
:   type: keyword


**`winlog.event_data.param6`**
:   type: keyword


**`winlog.event_data.param7`**
:   type: keyword


**`winlog.event_data.param8`**
:   type: keyword


**`winlog.event_id`**
:   The event identifier. The value is specific to the source of the event.

type: keyword

required: True


**`winlog.keywords`**
:   The keywords are used to classify an event.

type: keyword

required: False


**`winlog.channel`**
:   The name of the channel from which this record was read. This value is one of the names from the `event_logs` collection in the configuration.

type: keyword

required: True


**`winlog.record_id`**
:   The record ID of the event log record. The first record written to an event log is record number 1, and other records are numbered sequentially. If the record number reaches the maximum value (232 for the Event Logging API and 264 for the Windows Event Log API), the next record number will be 0.

type: keyword

required: True


**`winlog.related_activity_id`**
:   A globally unique identifier that identifies the activity to which control was transferred to. The related events would then have this identifier as their `activity_id` identifier.

type: keyword

required: False


**`winlog.opcode`**
:   The opcode defined in the event. Task and opcode are typically used to identify the location in the application from where the event was logged.

type: keyword

required: False


**`winlog.provider_guid`**
:   A globally unique identifier that identifies the provider that logged the event.

type: keyword

required: False


**`winlog.process.pid`**
:   The process_id of the Client Server Runtime Process.

type: long

required: False


**`winlog.provider_name`**
:   The source of the event log record (the application or service that logged the record).

type: keyword

required: True


**`winlog.task`**
:   The task defined in the event. Task and opcode are typically used to identify the location in the application from where the event was logged. The category used by the Event Logging API (on pre Windows Vista operating systems) is written to this field.

type: keyword

required: False


**`winlog.time_created`**
:   The event creation time.

type: date

required: False


**`winlog.trustAttribute`**
:   The decimal value of attributes for new trust created to a domain.

type: keyword

required: False


**`winlog.trustDirection`**
:   The direction of new trust created to a domain. Possible values are `TRUST_DIRECTION_DISABLED`, `TRUST_DIRECTION_INBOUND`, `TRUST_DIRECTION_OUTBOUND` and `TRUST_DIRECTION_BIDIRECTIONAL`

type: keyword

required: False


**`winlog.trustType`**
:   The account name that was added, modified or deleted in the event. Possible values are `TRUST_TYPE_DOWNLEVEL`, `TRUST_TYPE_UPLEVEL`, `TRUST_TYPE_MIT` and `TRUST_TYPE_DCE`

type: keyword

required: False


**`winlog.process.thread.id`**
:   type: long

required: False


**`winlog.user_data`**
:   The event specific data. This field is mutually exclusive with `event_data`.

type: object

required: False


**`winlog.user.identifier`**
:   The Windows security identifier (SID) of the account associated with this event. If Winlogbeat cannot resolve the SID to a name, then the `user.name`, `user.domain`, and `user.type` fields will be omitted from the event. If you discover Winlogbeat not resolving SIDs, review the log for clues as to what the problem may be.

type: keyword

example: S-1-5-21-3541430928-2051711210-1391384369-1001

required: False


**`winlog.user.name`**
:   Name of the user associated with this event.

type: keyword


**`winlog.user.domain`**
:   The domain that the account associated with this event is a member of.

type: keyword

required: False


**`winlog.user.type`**
:   The type of account associated with this event.

type: keyword

required: False


**`winlog.version`**
:   The version number of the event’s definition.

type: long

required: False


