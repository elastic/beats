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
=======
**`winlog.activity_id_name`**
:   The name of the activity that is associated with the activity_id. This is typically used to provide a human-readable name for the activity.

    type: keyword

    required: False


>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))
**`winlog.channel`**
:   Used to enable special event processing. Channel values below 16 are reserved for use by Microsoft to enable special treatment by the ETW runtime. Channel values 16 and above will be ignored by the ETW runtime (treated the same as channel 0) and can be given user-defined semantics.

    type: keyword

    required: False


**`winlog.event_data`**
:   The event-specific data. The content of this object is specific to any provider and event.

    type: object

    required: False


**`winlog.flags`**
<<<<<<< HEAD
:   Flags that provide information about the event such as the type of session it was logged to and if the event contains extended data.
=======
:   Flags that provide information about the event such as the type of session it was logged to and if the event contains extended data. This field is a list of flags, each flag is a string that represents a specific flag.

    type: keyword

    required: False


**`winlog.flags_raw`**
:   The bitmap of flags that provide information about the event such as the type of session it was logged to and if the event contains extended data.
>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))

    type: keyword

    required: False


**`winlog.keywords`**
<<<<<<< HEAD
:   The keywords are used to indicate an event's membership in a set of event categories.
=======
:   The keywords defined in the event. Keywords are used to indicate an event's membership in a set of event categories. This keywords are a list of keywords, each keyword is a string that represents a specific keyword.

    type: keyword

    required: False


**`winlog.keywords_raw`**
:   The bitmap of keywords that are used to indicate an event's membership in a set of event categories.
>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))

    type: keyword

    required: False


**`winlog.level`**
:   Level of severity. Level values 0 through 5 are defined by Microsoft. Level values 6 through 15 are reserved. Level values 16 through 255 can be defined by the event provider.

    type: keyword

    required: False


<<<<<<< HEAD
=======
**`winlog.level_raw`**
:   Numeric value of the level of severity. Level values 0 through 5 are defined by Microsoft. Level values 6 through 15 are reserved. Level values 16 through 255 can be defined by the event provider.

    type: long

    required: False


>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))
**`winlog.opcode`**
:   The opcode defined in the event. Task and opcode are typically used to identify the location in the application from where the event was logged.

    type: keyword

    required: False


<<<<<<< HEAD
=======
**`winlog.opcode_raw`**
:   Numeric value of the opcode defined in the event. This is used to identify the location in the application from where the event was logged.

    type: long

    required: False


>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))
**`winlog.process_id`**
:   Identifies the process that generated the event.

    type: keyword

    required: False


<<<<<<< HEAD
=======
**`winlog.provider`**
:   The source of the event log record (the application or service that logged the record).

    type: keyword

    required: False


>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))
**`winlog.provider_guid`**
:   A globally unique identifier that identifies the provider that logged the event.

    type: keyword

    required: False


<<<<<<< HEAD
**`winlog.provider_name`**
:   The source of the event log record (the application or service that logged the record).
=======
**`winlog.provider_message`**
:   The message that is associated with the provider. This is typically used to provide a human-readable name for the provider.

    type: keyword

    required: False


**`winlog.related_activity_id_name`**
:   The name of the related activity.
>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))

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


<<<<<<< HEAD
=======
**`winlog.task_raw`**
:   Numeric value of the task defined in the event. This is used to identify the location in the application from where the event was logged.

    type: long

    required: False


>>>>>>> 7fbc29824 ([docs automation] Update `generate_fields_docs.py` to add applies_to badges (#45772))
**`winlog.thread_id`**
:   Identifies the thread that generated the event.

    type: keyword

    required: False


**`winlog.version`**
:   Specify the version of a manifest-based event.

    type: long

    required: False


