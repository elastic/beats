---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-auditd.html
---

# Auditd fields [exported-fields-auditd]

Module for parsing auditd logs.

**`user.terminal`**
:   Terminal or tty device on which the user is performing the observed activity.

type: keyword


**`user.audit.id`**
:   One or multiple unique identifiers of the user.

type: keyword


**`user.audit.name`**
:   Short name or login of the user.

type: keyword

example: albert


**`user.audit.group.id`**
:   Unique identifier for the group on the system/platform.

type: keyword


**`user.audit.group.name`**
:   Name of the group.

type: keyword


**`user.filesystem.id`**
:   One or multiple unique identifiers of the user.

type: keyword


**`user.filesystem.name`**
:   Short name or login of the user.

type: keyword

example: albert


**`user.filesystem.group.id`**
:   Unique identifier for the group on the system/platform.

type: keyword


**`user.filesystem.group.name`**
:   Name of the group.

type: keyword


**`user.owner.id`**
:   One or multiple unique identifiers of the user.

type: keyword


**`user.owner.name`**
:   Short name or login of the user.

type: keyword

example: albert


**`user.owner.group.id`**
:   Unique identifier for the group on the system/platform.

type: keyword


**`user.owner.group.name`**
:   Name of the group.

type: keyword


**`user.saved.id`**
:   One or multiple unique identifiers of the user.

type: keyword


**`user.saved.name`**
:   Short name or login of the user.

type: keyword

example: albert


**`user.saved.group.id`**
:   Unique identifier for the group on the system/platform.

type: keyword


**`user.saved.group.name`**
:   Name of the group.

type: keyword



## auditd [_auditd]

Fields from the auditd logs.


## log [_log_2]

Fields from the Linux audit log. Not all fields are documented here because they are dynamic and vary by audit event type.

**`auditd.log.old_auid`**
:   For login events this is the old audit ID used for the user prior to this login.


**`auditd.log.new_auid`**
:   For login events this is the new audit ID. The audit ID can be used to trace future events to the user even if their identity changes (like becoming root).


**`auditd.log.old_ses`**
:   For login events this is the old session ID used for the user prior to this login.


**`auditd.log.new_ses`**
:   For login events this is the new session ID. It can be used to tie a user to future events by session ID.


**`auditd.log.sequence`**
:   The audit event sequence number.

type: long


**`auditd.log.items`**
:   The number of items in an event.


**`auditd.log.item`**
:   The item field indicates which item out of the total number of items. This number is zero-based; a value of 0 means it is the first item.


**`auditd.log.tty`**
:   type: keyword


**`auditd.log.a0`**
:   The first argument to the system call.


**`auditd.log.addr`**
:   type: ip


**`auditd.log.rport`**
:   type: long


**`auditd.log.laddr`**
:   type: ip


**`auditd.log.lport`**
:   type: long


**`auditd.log.acct`**
:   type: alias

alias to: user.name


**`auditd.log.pid`**
:   type: alias

alias to: process.pid


**`auditd.log.ppid`**
:   type: alias

alias to: process.parent.pid


**`auditd.log.res`**
:   type: alias

alias to: event.outcome


**`auditd.log.record_type`**
:   type: alias

alias to: event.action


**`auditd.log.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`auditd.log.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`auditd.log.geoip.location`**
:   type: alias

alias to: source.geo.location


**`auditd.log.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`auditd.log.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`auditd.log.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code


**`auditd.log.arch`**
:   type: alias

alias to: host.architecture


**`auditd.log.gid`**
:   type: alias

alias to: user.group.id


**`auditd.log.uid`**
:   type: alias

alias to: user.id


**`auditd.log.agid`**
:   type: alias

alias to: user.audit.group.id


**`auditd.log.auid`**
:   type: alias

alias to: user.audit.id


**`auditd.log.fsgid`**
:   type: alias

alias to: user.filesystem.group.id


**`auditd.log.fsuid`**
:   type: alias

alias to: user.filesystem.id


**`auditd.log.egid`**
:   type: alias

alias to: user.effective.group.id


**`auditd.log.euid`**
:   type: alias

alias to: user.effective.id


**`auditd.log.sgid`**
:   type: alias

alias to: user.saved.group.id


**`auditd.log.suid`**
:   type: alias

alias to: user.saved.id


**`auditd.log.ogid`**
:   type: alias

alias to: user.owner.group.id


**`auditd.log.ouid`**
:   type: alias

alias to: user.owner.id


**`auditd.log.comm`**
:   type: alias

alias to: process.name


**`auditd.log.exe`**
:   type: alias

alias to: process.executable


**`auditd.log.terminal`**
:   type: alias

alias to: user.terminal


**`auditd.log.msg`**
:   type: alias

alias to: message


**`auditd.log.src`**
:   type: alias

alias to: source.address


**`auditd.log.dst`**
:   type: alias

alias to: destination.address


