---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-zookeeper.html
---

# ZooKeeper fields [exported-fields-zookeeper]

ZooKeeper Module


## zookeeper [_zookeeper]


## audit [_audit_6]

ZooKeeper Audit logs.

**`zookeeper.audit.session`**
:   Client session id

type: keyword


**`zookeeper.audit.znode`**
:   Path of the znode

type: keyword


**`zookeeper.audit.znode_type`**
:   Type of znode in case of creation operation

type: keyword


**`zookeeper.audit.acl`**
:   String representation of znode ACL like cdrwa(create, delete,read, write, admin). This is logged only for setAcl operation

type: keyword


**`zookeeper.audit.result`**
:   Result of the operation. Possible values are (success/failure/invoked). Result "invoked" is used for serverStop operation because stop is logged before ensuring that server actually stopped.

type: keyword


**`zookeeper.audit.user`**
:   Comma separated list of users who are associate with a client session

type: keyword



## log [_log_14]

ZooKeeper logs.

