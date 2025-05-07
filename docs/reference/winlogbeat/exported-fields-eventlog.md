---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/exported-fields-eventlog.html
---

# Legacy Winlogbeat alias fields [exported-fields-eventlog]

Field aliases based on Winlogbeat 6.x that point to the fields for this version of Winlogbeat. These are added to the index template when `migration.6_to_7.enable: true` is set in the configuration.

**`activity_id`**
:   type: alias

alias to: winlog.activity_id


**`computer_name`**
:   type: alias

alias to: winlog.computer_name


**`event_id`**
:   type: alias

alias to: winlog.event_id


**`keywords`**
:   type: alias

alias to: winlog.keywords


**`log_name`**
:   type: alias

alias to: winlog.channel


**`message_error`**
:   type: alias

alias to: error.message


**`record_number`**
:   type: alias

alias to: winlog.record_id


**`related_activity_id`**
:   type: alias

alias to: winlog.related_activity_id


**`opcode`**
:   type: alias

alias to: winlog.opcode


**`provider_guid`**
:   type: alias

alias to: winlog.provider_guid


**`process_id`**
:   type: alias

alias to: winlog.process.pid


**`source_name`**
:   type: alias

alias to: winlog.provider_name


**`task`**
:   type: alias

alias to: winlog.task


**`thread_id`**
:   type: alias

alias to: winlog.process.thread.id


**`user.identifier`**
:   type: alias

alias to: winlog.user.identifier


**`user.type`**
:   type: alias

alias to: winlog.user.type


**`version`**
:   type: alias

alias to: winlog.version


**`xml`**
:   type: alias

alias to: event.original


