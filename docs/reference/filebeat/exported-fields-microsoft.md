---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-microsoft.html
---

# Microsoft fields [exported-fields-microsoft]

Microsoft Module


## microsoft.defender_atp [_microsoft_defender_atp]

Module for ingesting Microsoft Defender ATP.

**`microsoft.defender_atp.lastUpdateTime`**
:   The date and time (in UTC) the alert was last updated.

type: date


**`microsoft.defender_atp.resolvedTime`**
:   The date and time in which the status of the alert was changed to *Resolved*.

type: date


**`microsoft.defender_atp.incidentId`**
:   The Incident ID of the Alert.

type: keyword


**`microsoft.defender_atp.investigationId`**
:   The Investigation ID related to the Alert.

type: keyword


**`microsoft.defender_atp.investigationState`**
:   The current state of the Investigation.

type: keyword


**`microsoft.defender_atp.assignedTo`**
:   Owner of the alert.

type: keyword


**`microsoft.defender_atp.status`**
:   Specifies the current status of the alert. Possible values are: *Unknown*, *New*, *InProgress* and *Resolved*.

type: keyword


**`microsoft.defender_atp.classification`**
:   Specification of the alert. Possible values are: *Unknown*, *FalsePositive*, *TruePositive*.

type: keyword


**`microsoft.defender_atp.determination`**
:   Specifies the determination of the alert. Possible values are: *NotAvailable*, *Apt*, *Malware*, *SecurityPersonnel*, *SecurityTesting*, *UnwantedSoftware*, *Other*.

type: keyword


**`microsoft.defender_atp.threatFamilyName`**
:   Threat family.

type: keyword


**`microsoft.defender_atp.rbacGroupName`**
:   User group related to the alert

type: keyword


**`microsoft.defender_atp.evidence.domainName`**
:   Domain name related to the alert

type: keyword


**`microsoft.defender_atp.evidence.ipAddress`**
:   IP address involved in the alert

type: ip


**`microsoft.defender_atp.evidence.aadUserId`**
:   ID of the user involved in the alert

type: keyword


**`microsoft.defender_atp.evidence.accountName`**
:   Username of the user involved in the alert

type: keyword


**`microsoft.defender_atp.evidence.entityType`**
:   The type of evidence

type: keyword


**`microsoft.defender_atp.evidence.userPrincipalName`**
:   Principal name of the user involved in the alert

type: keyword



## microsoft.m365_defender [_microsoft_m365_defender]

Module for ingesting Microsoft Defender ATP.

**`microsoft.m365_defender.incidentId`**
:   Unique identifier to represent the incident.

type: keyword


**`microsoft.m365_defender.redirectIncidentId`**
:   Only populated in case an incident is being grouped together with another incident, as part of the incident processing logic.

type: keyword


**`microsoft.m365_defender.incidentName`**
:   Name of the Incident.

type: keyword


**`microsoft.m365_defender.determination`**
:   Specifies the determination of the incident. The property values are: NotAvailable, Apt, Malware, SecurityPersonnel, SecurityTesting, UnwantedSoftware, Other.

type: keyword


**`microsoft.m365_defender.investigationState`**
:   The current state of the Investigation.

type: keyword


**`microsoft.m365_defender.assignedTo`**
:   Owner of the alert.

type: keyword


**`microsoft.m365_defender.tags`**
:   Array of custom tags associated with an incident, for example to flag a group of incidents with a common characteristic.

type: keyword


**`microsoft.m365_defender.status`**
:   Specifies the current status of the alert. Possible values are: *Unknown*, *New*, *InProgress* and *Resolved*.

type: keyword


**`microsoft.m365_defender.classification`**
:   Specification of the alert. Possible values are: *Unknown*, *FalsePositive*, *TruePositive*.

type: keyword


**`microsoft.m365_defender.alerts.incidentId`**
:   Unique identifier to represent the incident this alert is associated with.

type: keyword


**`microsoft.m365_defender.alerts.resolvedTime`**
:   Time when alert was resolved.

type: date


**`microsoft.m365_defender.alerts.status`**
:   Categorize alerts (as New, Active, or Resolved).

type: keyword


**`microsoft.m365_defender.alerts.severity`**
:   The severity of the related alert.

type: keyword


**`microsoft.m365_defender.alerts.creationTime`**
:   Time when alert was first created.

type: date


**`microsoft.m365_defender.alerts.lastUpdatedTime`**
:   Time when alert was last updated.

type: date


**`microsoft.m365_defender.alerts.investigationId`**
:   The automated investigation id triggered by this alert.

type: keyword


**`microsoft.m365_defender.alerts.userSid`**
:   The SID of the related user

type: keyword


**`microsoft.m365_defender.alerts.detectionSource`**
:   The service that initially detected the threat.

type: keyword


**`microsoft.m365_defender.alerts.classification`**
:   The specification for the incident. The property values are: Unknown, FalsePositive, TruePositive or null.

type: keyword


**`microsoft.m365_defender.alerts.investigationState`**
:   Information on the investigation’s current status.

type: keyword


**`microsoft.m365_defender.alerts.determination`**
:   Specifies the determination of the incident. The property values are: NotAvailable, Apt, Malware, SecurityPersonnel, SecurityTesting, UnwantedSoftware, Other or null

type: keyword


**`microsoft.m365_defender.alerts.assignedTo`**
:   Owner of the incident, or null if no owner is assigned.

type: keyword


**`microsoft.m365_defender.alerts.actorName`**
:   The activity group, if any, the associated with this alert.

type: keyword


**`microsoft.m365_defender.alerts.threatFamilyName`**
:   Threat family associated with this alert.

type: keyword


**`microsoft.m365_defender.alerts.mitreTechniques`**
:   The attack techniques, as aligned with the MITRE ATT&CK™ framework.

type: keyword


**`microsoft.m365_defender.alerts.entities.entityType`**
:   Entities that have been identified to be part of, or related to, a given alert. The properties values are: User, Ip, Url, File, Process, MailBox, MailMessage, MailCluster, Registry.

type: keyword


**`microsoft.m365_defender.alerts.entities.accountName`**
:   Account name of the related user.

type: keyword


**`microsoft.m365_defender.alerts.entities.mailboxDisplayName`**
:   The display name of the related mailbox.

type: keyword


**`microsoft.m365_defender.alerts.entities.mailboxAddress`**
:   The mail address of the related mailbox.

type: keyword


**`microsoft.m365_defender.alerts.entities.clusterBy`**
:   A list of metadata if the entityType is MailCluster.

type: keyword


**`microsoft.m365_defender.alerts.entities.sender`**
:   The sender for the related email message.

type: keyword


**`microsoft.m365_defender.alerts.entities.recipient`**
:   The recipient for the related email message.

type: keyword


**`microsoft.m365_defender.alerts.entities.subject`**
:   The subject for the related email message.

type: keyword


**`microsoft.m365_defender.alerts.entities.deliveryAction`**
:   The delivery status for the related email message.

type: keyword


**`microsoft.m365_defender.alerts.entities.securityGroupId`**
:   The Security Group ID for the user related to the email message.

type: keyword


**`microsoft.m365_defender.alerts.entities.securityGroupName`**
:   The Security Group Name for the user related to the email message.

type: keyword


**`microsoft.m365_defender.alerts.entities.registryHive`**
:   Reference to which Hive in registry the event is related to, if eventType is registry. Example: HKEY_LOCAL_MACHINE.

type: keyword


**`microsoft.m365_defender.alerts.entities.registryKey`**
:   Reference to the related registry key to the event.

type: keyword


**`microsoft.m365_defender.alerts.entities.registryValueType`**
:   Value type of the registry key/value pair related to the event.

type: keyword


**`microsoft.m365_defender.alerts.entities.deviceId`**
:   The unique ID of the device related to the event.

type: keyword


**`microsoft.m365_defender.alerts.entities.ipAddress`**
:   The related IP address to the event.

type: keyword


**`microsoft.m365_defender.alerts.devices`**
:   The devices related to the investigation.

type: flattened


