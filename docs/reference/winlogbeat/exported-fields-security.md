---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/exported-fields-security.html
---

# Security module fields [exported-fields-security]

These are the event fields specific to the module for the Security log.


## winlog.logon [_winlog_logon]

Data related to a Windows logon.

**`winlog.logon.type`**
:   Logon type name. This is the descriptive version of the `winlog.event_data.LogonType` ordinal. This is an enrichment added by the Security module.

type: keyword

example: RemoteInteractive


**`winlog.logon.id`**
:   Logon ID that can be used to associate this logon with other events related to the same logon session.

type: keyword


**`winlog.logon.failure.reason`**
:   The reason the logon failed.

type: keyword


**`winlog.logon.failure.status`**
:   The reason the logon failed. This is textual description based on the value of the hexadecimal `Status` field.

type: keyword


**`winlog.logon.failure.sub_status`**
:   Additional information about the logon failure. This is a textual description based on the value of the hexidecimal `SubStatus` field.

type: keyword


