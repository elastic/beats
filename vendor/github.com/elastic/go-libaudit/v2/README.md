# go-libaudit

[![Build Status](https://beats-ci.elastic.co/job/Library/job/go-libaudit-mbp/job/master/badge/icon)](https://beats-ci.elastic.co/job/Library/job/go-libaudit-mbp/job/master/)
[![Go Documentation](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)][godocs]

[travis]: http://travis-ci.org/elastic/go-libaudit
[godocs]: http://godoc.org/github.com/elastic/go-libaudit

go-libaudit is a library for Go (golang) for communicating with the Linux Audit
Framework. The Linux Audit Framework provides system call auditing in the kernel
and logs the events to user-space using netlink sockets. This library
facilitates user-space applications that want to receive audit events.

## Installation and Usage

Package documentation can be found on [GoDoc][godocs].

Installation can be done with a normal `go get`:

```
$ go get github.com/elastic/go-libaudit
```

go-libaudit has two example applications that you can use to try the library.
The first is _audit_ which registers to receive audit events from the kernel
and outputs the data it receives to stdout. The system's `auditd` process
should be stopped first.

```
$ go install github.com/elastic/go-libaudit/cmd/audit
$ sudo $GOPATH/bin/audit -d
```

The second is _auparse_ which parses the log files from the Linux auditd
process or the output of the _audit_ example command. It combines related log
messages that are a part of the same event.

```
$ go install github.com/elastic/go-libaudit/cmd/auparse
$ sudo cat /var/log/audit/audit.log | auparse
---
type=CRED_ACQ msg=audit(1481077334.302:545): pid=1444 uid=0 auid=1000 ses=4 subj=unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023 msg='op=PAM:setcred grantors=pam_env,pam_unix acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/1 res=success'
---
type=USER_START msg=audit(1481077334.303:546): pid=1444 uid=0 auid=1000 ses=4 subj=unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023 msg='op=PAM:session_open grantors=pam_keyinit,pam_limits acct="root" exe="/usr/bin/sudo" hostname=? addr=? terminal=/dev/pts/1 res=success'
---
type=SYSCALL msg=audit(1481077334.304:547): arch=c000003e syscall=59 success=yes exit=0 a0=7f683953a5d8 a1=7f683953fd38 a2=7f6839543a90 a3=6 items=2 ppid=1444 pid=1445 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts1 ses=4 comm="su" exe="/usr/bin/su" subj=unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023 key=(null)
type=EXECVE msg=audit(1481077334.304:547): argc=1 a0="su"
type=CWD msg=audit(1481077334.304:547):  cwd="/home/andrew_kroh"
type=PATH msg=audit(1481077334.304:547): item=0 name="/bin/su" inode=5026 dev=08:01 mode=0104755 ouid=0 ogid=0 rdev=00:00 obj=system_u:object_r:su_exec_t:s0 objtype=NORMAL
type=PATH msg=audit(1481077334.304:547): item=1 name="/lib64/ld-linux-x86-64.so.2" inode=16778495 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 obj=system_u:object_r:ld_so_t:s0 objtype=NORMAL
```

It supports outputting the messages as plain text (default), JSON, or YAML by
using `-format=yaml` for example.

```
$ sudo cat /var/log/audit/audit.log | auparse -format=json
---
{"@timestamp":"2016-12-07 02:22:14.302 +0000 UTC","acct":"root","auid":"1000","exe":"/usr/bin/sudo","grantors":"pam_env,pam_unix","op":"PAM:setcred","pid":"1444","raw_msg":"audit(1481077334.302:545): pid=1444 uid=0 auid=1000 ses=4 subj=unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023 msg='op=PAM:setcred grantors=pam_env,pam_unix acct=\"root\" exe=\"/usr/bin/sudo\" hostname=? addr=? terminal=/dev/pts/1 res=success'","record_type":"CRED_ACQ","result":"success","sequence":"545","ses":"4","subj_category":"c0.c1023","subj_domain":"unconfined_t","subj_level":"s0-s0","subj_role":"unconfined_r","subj_user":"unconfined_u","terminal":"/dev/pts/1","uid":"0"}
---
{"@timestamp":"2016-12-07 02:22:14.303 +0000 UTC","acct":"root","auid":"1000","exe":"/usr/bin/sudo","grantors":"pam_keyinit,pam_limits","op":"PAM:session_open","pid":"1444","raw_msg":"audit(1481077334.303:546): pid=1444 uid=0 auid=1000 ses=4 subj=unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023 msg='op=PAM:session_open grantors=pam_keyinit,pam_limits acct=\"root\" exe=\"/usr/bin/sudo\" hostname=? addr=? terminal=/dev/pts/1 res=success'","record_type":"USER_START","result":"success","sequence":"546","ses":"4","subj_category":"c0.c1023","subj_domain":"unconfined_t","subj_level":"s0-s0","subj_role":"unconfined_r","subj_user":"unconfined_u","terminal":"/dev/pts/1","uid":"0"}
---
{"@timestamp":"2016-12-07 02:22:14.304 +0000 UTC","a0":"7f683953a5d8","a1":"7f683953fd38","a2":"7f6839543a90","a3":"6","arch":"x86_64","auid":"1000","comm":"su","egid":"0","euid":"0","exe":"/usr/bin/su","exit":"0","fsgid":"0","fsuid":"0","gid":"0","items":"2","pid":"1445","ppid":"1444","raw_msg":"audit(1481077334.304:547): arch=c000003e syscall=59 success=yes exit=0 a0=7f683953a5d8 a1=7f683953fd38 a2=7f6839543a90 a3=6 items=2 ppid=1444 pid=1445 auid=1000 uid=0 gid=0 euid=0 suid=0 fsuid=0 egid=0 sgid=0 fsgid=0 tty=pts1 ses=4 comm=\"su\" exe=\"/usr/bin/su\" subj=unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023 key=(null)","record_type":"SYSCALL","result":"success","sequence":"547","ses":"4","sgid":"0","subj_category":"c0.c1023","subj_domain":"unconfined_t","subj_level":"s0-s0","subj_role":"unconfined_r","subj_user":"unconfined_u","suid":"0","syscall":"execve","tty":"pts1","uid":"0"}
{"@timestamp":"2016-12-07 02:22:14.304 +0000 UTC","a0":"su","argc":"1","raw_msg":"audit(1481077334.304:547): argc=1 a0=\"su\"","record_type":"EXECVE","sequence":"547"}
{"@timestamp":"2016-12-07 02:22:14.304 +0000 UTC","cwd":"/home/andrew_kroh","raw_msg":"audit(1481077334.304:547):  cwd=\"/home/andrew_kroh\"","record_type":"CWD","sequence":"547"}
{"@timestamp":"2016-12-07 02:22:14.304 +0000 UTC","dev":"08:01","inode":"5026","item":"0","mode":"0104755","name":"/bin/su","obj_domain":"su_exec_t","obj_level":"s0","obj_role":"object_r","obj_user":"system_u","objtype":"NORMAL","ogid":"0","ouid":"0","raw_msg":"audit(1481077334.304:547): item=0 name=\"/bin/su\" inode=5026 dev=08:01 mode=0104755 ouid=0 ogid=0 rdev=00:00 obj=system_u:object_r:su_exec_t:s0 objtype=NORMAL","rdev":"00:00","record_type":"PATH","sequence":"547"}
{"@timestamp":"2016-12-07 02:22:14.304 +0000 UTC","dev":"08:01","inode":"16778495","item":"1","mode":"0100755","name":"/lib64/ld-linux-x86-64.so.2","obj_domain":"ld_so_t","obj_level":"s0","obj_role":"object_r","obj_user":"system_u","objtype":"NORMAL","ogid":"0","ouid":"0","raw_msg":"audit(1481077334.304:547): item=1 name=\"/lib64/ld-linux-x86-64.so.2\" inode=16778495 dev=08:01 mode=0100755 ouid=0 ogid=0 rdev=00:00 obj=system_u:object_r:ld_so_t:s0 objtype=NORMAL","rdev":"00:00","record_type":"PATH","sequence":"547"}
```

To normalize and interpret the messages, use the `-i` flag for "interpret". This
adds a category to the event and creates the `actor`, `action`, `thing`, and
`how` fields based on data from the event. By default it will resolve UID and
GID values to their names (use `-id=false` to disable this).

```
$ sudo cat /var/log/audit/audit.log | auparse -format=yaml -i
---
timestamp: 2016-12-07T02:22:14.302Z
sequence: 545
category: user-login
record_type: cred_acq
result: success
session: "4"
summary:
  actor:
    primary: vagrant
    secondary: root
  action: acquired-credentials
  object:
    type: user-session
    primary: /dev/pts/1
  how: /usr/bin/sudo
user:
  ids:
    auid: "1000"
    uid: "0"
  names:
    auid: vagrant
    uid: root
  selinux:
    category: c0.c1023
    domain: unconfined_t
    level: s0-s0
    role: unconfined_r
    user: unconfined_u
process:
  pid: "1444"
  exe: /usr/bin/sudo
data:
  acct: root
  grantors: pam_env,pam_unix
  op: PAM:setcred
  terminal: /dev/pts/1
ecs:
  event:
    category:
    - authentication
    type:
    - info
  user:
    name: vagrant
    id: "1000"
    effective:
      name: root
      id: "0"
    target: {}
    changes: {}
  group: {}

---
timestamp: 2016-12-07T02:22:14.303Z
sequence: 546
category: user-login
record_type: user_start
result: success
session: "4"
summary:
  actor:
    primary: vagrant
    secondary: root
  action: started-session
  object:
    type: user-session
    primary: /dev/pts/1
  how: /usr/bin/sudo
user:
  ids:
    auid: "1000"
    uid: "0"
  names:
    auid: vagrant
    uid: root
  selinux:
    category: c0.c1023
    domain: unconfined_t
    level: s0-s0
    role: unconfined_r
    user: unconfined_u
process:
  pid: "1444"
  exe: /usr/bin/sudo
data:
  acct: root
  grantors: pam_keyinit,pam_limits
  op: PAM:session_open
  terminal: /dev/pts/1
ecs:
  event:
    category:
    - authentication
    type:
    - info
  user:
    name: vagrant
    id: "1000"
    effective:
      name: root
      id: "0"
    target: {}
    changes: {}
  group: {}

---
timestamp: 2016-12-07T02:22:14.304Z
sequence: 547
category: audit-rule
record_type: syscall
result: success
session: "4"
summary:
  actor:
    primary: vagrant
    secondary: root
  action: executed
  object:
    type: file
    primary: /bin/su
  how: /usr/bin/su
user:
  ids:
    auid: "1000"
    egid: "0"
    euid: "0"
    fsgid: "0"
    fsuid: "0"
    gid: "0"
    sgid: "0"
    suid: "0"
    uid: "0"
  names:
    auid: vagrant
    egid: root
    euid: root
    fsgid: root
    fsuid: root
    gid: root
    sgid: root
    suid: root
    uid: root
  selinux:
    category: c0.c1023
    domain: unconfined_t
    level: s0-s0
    role: unconfined_r
    user: unconfined_u
process:
  pid: "1445"
  ppid: "1444"
  name: su
  exe: /usr/bin/su
  cwd: /home/andrew_kroh
  args:
  - su
file:
  path: /bin/su
  device: "00:00"
  inode: "5026"
  mode: "0755"
  uid: "0"
  gid: "0"
  owner: root
  group: root
  selinux:
    domain: su_exec_t
    level: s0
    role: object_r
    user: system_u
data:
  a0: 7f683953a5d8
  a1: 7f683953fd38
  a2: 7f6839543a90
  a3: "6"
  arch: x86_64
  argc: "1"
  exit: "0"
  syscall: execve
  tty: pts1
paths:
- dev: "08:01"
  inode: "5026"
  item: "0"
  mode: "0104755"
  name: /bin/su
  obj_domain: su_exec_t
  obj_level: s0
  obj_role: object_r
  obj_user: system_u
  objtype: NORMAL
  ogid: "0"
  ouid: "0"
  rdev: "00:00"
- dev: "08:01"
  inode: "16778495"
  item: "1"
  mode: "0100755"
  name: /lib64/ld-linux-x86-64.so.2
  obj_domain: ld_so_t
  obj_level: s0
  obj_role: object_r
  obj_user: system_u
  objtype: NORMAL
  ogid: "0"
  ouid: "0"
  rdev: "00:00"
ecs:
  event:
    category:
    - process
    type:
    - start
  user:
    effective: {}
    target: {}
    changes: {}
  group: {}
```

## ECS compatibility

This currently provides [Elastic Common Schema (ECS) 1.8](https://www.elastic.co/guide/en/ecs/current/index.html) categorization support for some of the more prominent or meaningful auditd events and syscalls.
