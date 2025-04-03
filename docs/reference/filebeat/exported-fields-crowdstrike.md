---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-crowdstrike.html
---

# Crowdstrike fields [exported-fields-crowdstrike]

Module for collecting Crowdstrike events.


## crowdstrike [_crowdstrike]

Fields for Crowdstrike Falcon event and alert data.


## metadata [_metadata_2]

Meta data fields for each event that include type and timestamp.

**`crowdstrike.metadata.eventType`**
:   DetectionSummaryEvent, FirewallMatchEvent, IncidentSummaryEvent, RemoteResponseSessionStartEvent, RemoteResponseSessionEndEvent, AuthActivityAuditEvent, or UserActivityAuditEvent

type: keyword


**`crowdstrike.metadata.eventCreationTime`**
:   The time this event occurred on the endpoint in UTC UNIX_MS format.

type: date


**`crowdstrike.metadata.offset`**
:   Offset number that tracks the location of the event in stream. This is used to identify unique detection events.

type: integer


**`crowdstrike.metadata.customerIDString`**
:   Customer identifier

type: keyword


**`crowdstrike.metadata.version`**
:   Schema version

type: keyword



## event [_event]

Event data fields for each event and alert.

**`crowdstrike.event.ProcessStartTime`**
:   The process start time in UTC UNIX_MS format.

type: date


**`crowdstrike.event.ProcessEndTime`**
:   The process termination time in UTC UNIX_MS format.

type: date


**`crowdstrike.event.ProcessId`**
:   Process ID related to the detection.

type: integer


**`crowdstrike.event.ParentProcessId`**
:   Parent process ID related to the detection.

type: integer


**`crowdstrike.event.ComputerName`**
:   Name of the computer where the detection occurred.

type: keyword


**`crowdstrike.event.UserName`**
:   User name associated with the detection.

type: keyword


**`crowdstrike.event.DetectName`**
:   Name of the detection.

type: keyword


**`crowdstrike.event.DetectDescription`**
:   Description of the detection.

type: keyword


**`crowdstrike.event.Severity`**
:   Severity score of the detection.

type: integer


**`crowdstrike.event.SeverityName`**
:   Severity score text.

type: keyword


**`crowdstrike.event.FileName`**
:   File name of the associated process for the detection.

type: keyword


**`crowdstrike.event.FilePath`**
:   Path of the executable associated with the detection.

type: keyword


**`crowdstrike.event.CommandLine`**
:   Executable path with command line arguments.

type: keyword


**`crowdstrike.event.SHA1String`**
:   SHA1 sum of the executable associated with the detection.

type: keyword


**`crowdstrike.event.SHA256String`**
:   SHA256 sum of the executable associated with the detection.

type: keyword


**`crowdstrike.event.MD5String`**
:   MD5 sum of the executable associated with the detection.

type: keyword


**`crowdstrike.event.MachineDomain`**
:   Domain for the machine associated with the detection.

type: keyword


**`crowdstrike.event.FalconHostLink`**
:   URL to view the detection in Falcon.

type: keyword


**`crowdstrike.event.SensorId`**
:   Unique ID associated with the Falcon sensor.

type: keyword


**`crowdstrike.event.DetectId`**
:   Unique ID associated with the detection.

type: keyword


**`crowdstrike.event.LocalIP`**
:   IP address of the host associated with the detection.

type: keyword


**`crowdstrike.event.MACAddress`**
:   MAC address of the host associated with the detection.

type: keyword


**`crowdstrike.event.Tactic`**
:   MITRE tactic category of the detection.

type: keyword


**`crowdstrike.event.Technique`**
:   MITRE technique category of the detection.

type: keyword


**`crowdstrike.event.Objective`**
:   Method of detection.

type: keyword


**`crowdstrike.event.PatternDispositionDescription`**
:   Action taken by Falcon.

type: keyword


**`crowdstrike.event.PatternDispositionValue`**
:   Unique ID associated with action taken.

type: integer


**`crowdstrike.event.PatternDispositionFlags`**
:   Flags indicating actions taken.

type: object


**`crowdstrike.event.State`**
:   Whether the incident summary is open and ongoing or closed.

type: keyword


**`crowdstrike.event.IncidentStartTime`**
:   Start time for the incident in UTC UNIX format.

type: date


**`crowdstrike.event.IncidentEndTime`**
:   End time for the incident in UTC UNIX format.

type: date


**`crowdstrike.event.FineScore`**
:   Score for incident.

type: float


**`crowdstrike.event.UserId`**
:   Email address or user ID associated with the event.

type: keyword


**`crowdstrike.event.UserIp`**
:   IP address associated with the user.

type: keyword


**`crowdstrike.event.OperationName`**
:   Event subtype.

type: keyword


**`crowdstrike.event.ServiceName`**
:   Service associated with this event.

type: keyword


**`crowdstrike.event.Success`**
:   Indicator of whether or not this event was successful.

type: boolean


**`crowdstrike.event.UTCTimestamp`**
:   Timestamp associated with this event in UTC UNIX format.

type: date


**`crowdstrike.event.AuditKeyValues`**
:   Fields that were changed in this event.

type: nested


**`crowdstrike.event.ExecutablesWritten`**
:   Detected executables written to disk by a process.

type: nested


**`crowdstrike.event.SessionId`**
:   Session ID of the remote response session.

type: keyword


**`crowdstrike.event.HostnameField`**
:   Host name of the machine for the remote session.

type: keyword


**`crowdstrike.event.StartTimestamp`**
:   Start time for the remote session in UTC UNIX format.

type: date


**`crowdstrike.event.EndTimestamp`**
:   End time for the remote session in UTC UNIX format.

type: date


**`crowdstrike.event.LateralMovement`**
:   Lateral movement field for incident.

type: long


**`crowdstrike.event.ParentImageFileName`**
:   Path to the parent process.

type: keyword


**`crowdstrike.event.ParentCommandLine`**
:   Parent process command line arguments.

type: keyword


**`crowdstrike.event.GrandparentImageFileName`**
:   Path to the grandparent process.

type: keyword


**`crowdstrike.event.GrandparentCommandLine`**
:   Grandparent process command line arguments.

type: keyword


**`crowdstrike.event.IOCType`**
:   CrowdStrike type for indicator of compromise.

type: keyword


**`crowdstrike.event.IOCValue`**
:   CrowdStrike value for indicator of compromise.

type: keyword


**`crowdstrike.event.CustomerId`**
:   Customer identifier.

type: keyword


**`crowdstrike.event.DeviceId`**
:   Device on which the event occurred.

type: keyword


**`crowdstrike.event.Ipv`**
:   Protocol for network request.

type: keyword


**`crowdstrike.event.ConnectionDirection`**
:   Direction for network connection.

type: keyword


**`crowdstrike.event.EventType`**
:   CrowdStrike provided event type.

type: keyword


**`crowdstrike.event.HostName`**
:   Host name of the local machine.

type: keyword


**`crowdstrike.event.ICMPCode`**
:   RFC2780 ICMP Code field.

type: keyword


**`crowdstrike.event.ICMPType`**
:   RFC2780 ICMP Type field.

type: keyword


**`crowdstrike.event.ImageFileName`**
:   File name of the associated process for the detection.

type: keyword


**`crowdstrike.event.PID`**
:   Associated process id for the detection.

type: long


**`crowdstrike.event.LocalAddress`**
:   IP address of local machine.

type: ip


**`crowdstrike.event.LocalPort`**
:   Port of local machine.

type: long


**`crowdstrike.event.RemoteAddress`**
:   IP address of remote machine.

type: ip


**`crowdstrike.event.RemotePort`**
:   Port of remote machine.

type: long


**`crowdstrike.event.RuleAction`**
:   Firewall rule action.

type: keyword


**`crowdstrike.event.RuleDescription`**
:   Firewall rule description.

type: keyword


**`crowdstrike.event.RuleFamilyID`**
:   Firewall rule family id.

type: keyword


**`crowdstrike.event.RuleGroupName`**
:   Firewall rule group name.

type: keyword


**`crowdstrike.event.RuleName`**
:   Firewall rule name.

type: keyword


**`crowdstrike.event.RuleId`**
:   Firewall rule id.

type: keyword


**`crowdstrike.event.MatchCount`**
:   Number of firewall rule matches.

type: long


**`crowdstrike.event.MatchCountSinceLastReport`**
:   Number of firewall rule matches since the last report.

type: long


**`crowdstrike.event.Timestamp`**
:   Firewall rule triggered timestamp.

type: date


**`crowdstrike.event.Flags.Audit`**
:   CrowdStrike audit flag.

type: boolean


**`crowdstrike.event.Flags.Log`**
:   CrowdStrike log flag.

type: boolean


**`crowdstrike.event.Flags.Monitor`**
:   CrowdStrike monitor flag.

type: boolean


**`crowdstrike.event.Protocol`**
:   CrowdStrike provided protocol.

type: keyword


**`crowdstrike.event.NetworkProfile`**
:   CrowdStrike network profile.

type: keyword


**`crowdstrike.event.PolicyName`**
:   CrowdStrike policy name.

type: keyword


**`crowdstrike.event.PolicyID`**
:   CrowdStrike policy id.

type: keyword


**`crowdstrike.event.Status`**
:   CrowdStrike status.

type: keyword


**`crowdstrike.event.TreeID`**
:   CrowdStrike tree id.

type: keyword


**`crowdstrike.event.Commands`**
:   Commands run in a remote session.

type: keyword


