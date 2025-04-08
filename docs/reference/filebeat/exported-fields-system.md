---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-system.html
---

# System fields [exported-fields-system]

Module for parsing system log files.


## system [_system]

Fields from the system log files.


## auth [_auth_2]

Fields from the Linux authorization logs.

**`system.auth.timestamp`**
:   type: alias

alias to: @timestamp


**`system.auth.hostname`**
:   type: alias

alias to: host.hostname


**`system.auth.program`**
:   type: alias

alias to: process.name


**`system.auth.pid`**
:   type: alias

alias to: process.pid


**`system.auth.message`**
:   type: alias

alias to: message


**`system.auth.user`**
:   type: alias

alias to: user.name


**`system.auth.ssh.method`**
:   The SSH authentication method. Can be one of "password" or "publickey".


**`system.auth.ssh.signature`**
:   The signature of the client public key.


**`system.auth.ssh.dropped_ip`**
:   The client IP from SSH connections that are open and immediately dropped.

type: ip


**`system.auth.ssh.event`**
:   The SSH event as found in the logs (Accepted, Invalid, Failed, etc.)

example: Accepted


**`system.auth.ssh.ip`**
:   type: alias

alias to: source.ip


**`system.auth.ssh.port`**
:   type: alias

alias to: source.port


**`system.auth.ssh.geoip.continent_name`**
:   type: alias

alias to: source.geo.continent_name


**`system.auth.ssh.geoip.country_iso_code`**
:   type: alias

alias to: source.geo.country_iso_code


**`system.auth.ssh.geoip.location`**
:   type: alias

alias to: source.geo.location


**`system.auth.ssh.geoip.region_name`**
:   type: alias

alias to: source.geo.region_name


**`system.auth.ssh.geoip.city_name`**
:   type: alias

alias to: source.geo.city_name


**`system.auth.ssh.geoip.region_iso_code`**
:   type: alias

alias to: source.geo.region_iso_code



## sudo [_sudo]

Fields specific to events created by the `sudo` command.

**`system.auth.sudo.error`**
:   The error message in case the sudo command failed.

example: user NOT in sudoers


**`system.auth.sudo.tty`**
:   The TTY where the sudo command is executed.


**`system.auth.sudo.pwd`**
:   The current directory where the sudo command is executed.


**`system.auth.sudo.user`**
:   The target user to which the sudo command is switching.

example: root


**`system.auth.sudo.command`**
:   The command executed via sudo.



## useradd [_useradd]

Fields specific to events created by the `useradd` command.

**`system.auth.useradd.home`**
:   The home folder for the new user.


**`system.auth.useradd.shell`**
:   The default shell for the new user.


**`system.auth.useradd.name`**
:   type: alias

alias to: user.name


**`system.auth.useradd.uid`**
:   type: alias

alias to: user.id


**`system.auth.useradd.gid`**
:   type: alias

alias to: group.id



## groupadd [_groupadd]

Fields specific to events created by the `groupadd` command.

**`system.auth.groupadd.name`**
:   type: alias

alias to: group.name


**`system.auth.groupadd.gid`**
:   type: alias

alias to: group.id



## syslog [_syslog_3]

Contains fields from the syslog system logs.

**`system.syslog.timestamp`**
:   type: alias

alias to: @timestamp


**`system.syslog.hostname`**
:   type: alias

alias to: host.hostname


**`system.syslog.program`**
:   type: alias

alias to: process.name


**`system.syslog.pid`**
:   type: alias

alias to: process.pid


**`system.syslog.message`**
:   type: alias

alias to: message


