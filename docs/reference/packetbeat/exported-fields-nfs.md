---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-nfs.html
---

# NFS fields [exported-fields-nfs]

NFS v4/3 specific event fields.

**`nfs.version`**
:   NFS protocol version number.

type: long


**`nfs.minor_version`**
:   NFS protocol minor version number.

type: long


**`nfs.tag`**
:   NFS v4 COMPOUND operation tag.


**`nfs.opcode`**
:   NFS operation name, or main operation name, in case of COMPOUND calls.


**`nfs.status`**
:   NFS operation reply status.



## rpc [_rpc]

ONC RPC specific event fields.

**`rpc.xid`**
:   RPC message transaction identifier.


**`rpc.status`**
:   RPC message reply status.


**`rpc.auth_flavor`**
:   RPC authentication flavor.


**`rpc.cred.uid`**
:   RPC caller’s user id, in case of auth-unix.

type: long


**`rpc.cred.gid`**
:   RPC caller’s group id, in case of auth-unix.

type: long


**`rpc.cred.gids`**
:   RPC caller’s secondary group ids, in case of auth-unix.


**`rpc.cred.stamp`**
:   Arbitrary ID which the caller machine may generate.

type: long


**`rpc.cred.machinename`**
:   The name of the caller’s machine.


**`rpc.call_size`**
:   RPC call size with argument.

type: alias

alias to: source.bytes


**`rpc.reply_size`**
:   RPC reply size with argument.

type: alias

alias to: destination.bytes


