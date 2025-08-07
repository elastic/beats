---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-winlog.html
---

% This file is generated! See scripts/generate_fields_docs.py

# Windows ETW fields [exported-fields-winlog]

Fields from the ETW input (Event Tracing for Windows).

## winlog [_winlog]

All fields specific to the Windows Event Tracing are defined here.

**`winlog.activity_id`**
:   A globally unique identifier that identifies the current activity. The events that are published with this identifier are part of the same activity.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.activity_id_name`**
:   The name of the activity that is associated with the activity_id. This is typically used to provide a human-readable name for the activity.

    type: keyword

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.channel`**
:   The channel that the event was logged to. The channel is a logical grouping of events that are logged by a provider. The channel is typically used to identify the type of events that are logged, such as security, application, or system events.

    type: keyword

    required: False


**`winlog.event_data`**
:   The event-specific data. The content of this object is specific to any provider and event.

    type: object

    required: False


**`winlog.flags`**
:   Flags that provide information about the event such as the type of session it was logged to and if the event contains extended data. This field is a list of flags, each flag is a string that represents a specific flag.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.flags_raw`**
:   The bitmap of flags that provide information about the event such as the type of session it was logged to and if the event contains extended data.

    type: keyword

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.keywords`**
:   The keywords defined in the event. Keywords are used to indicate an event's membership in a set of event categories. This keywords are a list of keywords, each keyword is a string that represents a specific keyword.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.keywords_raw`**
:   The bitmap of keywords that are used to indicate an event's membership in a set of event categories.

    type: keyword

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.level`**
:   Level of severity. Level values 0 through 5 are defined by Microsoft. Level values 6 through 15 are reserved. Level values 16 through 255 can be defined by the event provider.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.level_raw`**
:   Numeric value of the level of severity. Level values 0 through 5 are defined by Microsoft. Level values 6 through 15 are reserved. Level values 16 through 255 can be defined by the event provider.

    type: long

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.opcode`**
:   The opcode defined in the event. Task and opcode are typically used to identify the location in the application from where the event was logged.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.opcode_raw`**
:   Numeric value of the opcode defined in the event. This is used to identify the location in the application from where the event was logged.

    type: long

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.process_id`**
:   Identifies the process that generated the event.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.provider`**
:   The source of the event log record (the application or service that logged the record).

    type: keyword

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.provider_guid`**
:   A globally unique identifier that identifies the provider that logged the event.

    type: keyword

    required: False


**`winlog.provider_message`**
:   The message that is associated with the provider. This is typically used to provide a human-readable name for the provider.

    type: keyword

    required: False
<<<<<<< HEAD


**`winlog.related_activity_id_name`**
:   The name of the related activity.

    type: keyword

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


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
<<<<<<< HEAD


**`winlog.task_raw`**
:   Numeric value of the task defined in the event. This is used to identify the location in the application from where the event was logged.

    type: long

    required: False
=======
>>>>>>> 112199148 ([9.0](backport #45772) [docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45815))


**`winlog.thread_id`**
:   Identifies the thread that generated the event.

    type: keyword

    required: False


**`winlog.version`**
:   Specify the version of a manifest-based event.

    type: long

    required: False


