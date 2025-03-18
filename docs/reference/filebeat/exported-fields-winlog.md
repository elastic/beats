---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-winlog.html
---

# Windows ETW fields [exported-fields-winlog]

Fields from the ETW input (Event Tracing for Windows).


## winlog [_winlog]

All fields specific to the Windows Event Tracing are defined here.

**`winlog.activity_id`**
:   A globally unique identifier that identifies the current activity. The events that are published with this identifier are part of the same activity.

type: keyword

required: False


**`winlog.channel`**
:   Used to enable special event processing. Channel values below 16 are reserved for use by Microsoft to enable special treatment by the ETW runtime. Channel values 16 and above will be ignored by the ETW runtime (treated the same as channel 0) and can be given user-defined semantics.

type: keyword

required: False


**`winlog.event_data`**
:   The event-specific data. The content of this object is specific to any provider and event.

type: object

required: False


**`winlog.flags`**
:   Flags that provide information about the event such as the type of session it was logged to and if the event contains extended data.

type: keyword

required: False


**`winlog.keywords`**
:   The keywords are used to indicate an event’s membership in a set of event categories.

type: keyword

required: False


**`winlog.level`**
:   Level of severity. Level values 0 through 5 are defined by Microsoft. Level values 6 through 15 are reserved. Level values 16 through 255 can be defined by the event provider.

type: keyword

required: False


**`winlog.opcode`**
:   The opcode defined in the event. Task and opcode are typically used to identify the location in the application from where the event was logged.

type: keyword

required: False


**`winlog.process_id`**
:   Identifies the process that generated the event.

type: keyword

required: False


**`winlog.provider_guid`**
:   A globally unique identifier that identifies the provider that logged the event.

type: keyword

required: False


**`winlog.provider_name`**
:   The source of the event log record (the application or service that logged the record).

type: keyword

required: False


**`winlog.session`**
:   Configured session to forward ETW events from providers to consumers.

type: keyword

required: False


**`winlog.severity`**
:   Human-readable level of severity.

type: keyword

required: False


**`winlog.task`**
:   The task defined in the event. Task and opcode are typically used to identify the location in the application from where the event was logged.

type: keyword

required: False


**`winlog.thread_id`**
:   Identifies the thread that generated the event.

type: keyword

required: False


**`winlog.version`**
:   Specify the version of a manifest-based event.

type: long

required: False


