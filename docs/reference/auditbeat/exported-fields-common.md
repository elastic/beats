---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/exported-fields-common.html
---

# Common fields [exported-fields-common]

Contains common fields available in all event types.


## file [_file]

File attributes.

**`file.setuid`**
:   Set if the file has the `setuid` bit set. Omitted otherwise.

type: boolean

example: True


**`file.setgid`**
:   Set if the file has the `setgid` bit set. Omitted otherwise.

type: boolean

example: True


**`file.origin`**
:   An array of strings describing a possible external origin for this file. For example, the URL it was downloaded from. Only supported in macOS, via the kMDItemWhereFroms attribute. Omitted if origin information is not available.

type: keyword


**`file.origin.text`**
:   This is an analyzed field that is useful for full text search on the origin data.

type: text



## selinux [_selinux_2]

The SELinux identity of the file.

**`file.selinux.user`**
:   The owner of the object.

type: keyword


**`file.selinux.role`**
:   The object’s SELinux role.

type: keyword


**`file.selinux.domain`**
:   The object’s SELinux domain or type.

type: keyword


**`file.selinux.level`**
:   The object’s SELinux level.

type: keyword

example: s0



## user [_user]

User information.


## audit [_audit]

Audit user information.

**`user.audit.id`**
:   Audit user ID.

type: keyword


**`user.audit.name`**
:   Audit user name.

type: keyword



## filesystem [_filesystem]

Filesystem user information.

**`user.filesystem.id`**
:   Filesystem user ID.

type: keyword


**`user.filesystem.name`**
:   Filesystem user name.

type: keyword



## group [_group]

Filesystem group information.

**`user.filesystem.group.id`**
:   Filesystem group ID.

type: keyword


**`user.filesystem.group.name`**
:   Filesystem group name.

type: keyword



## saved [_saved]

Saved user information.

**`user.saved.id`**
:   Saved user ID.

type: keyword


**`user.saved.name`**
:   Saved user name.

type: keyword



## group [_group_2]

Saved group information.

**`user.saved.group.id`**
:   Saved group ID.

type: keyword


**`user.saved.group.name`**
:   Saved group name.

type: keyword


