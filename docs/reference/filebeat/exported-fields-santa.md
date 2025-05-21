---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-santa.html
---

# Google Santa fields [exported-fields-santa]

Santa Module


## santa [_santa]

**`santa.action`**
:   Action

type: keyword

example: EXEC


**`santa.decision`**
:   Decision that santad took.

type: keyword

example: ALLOW


**`santa.reason`**
:   Reason for the decsision.

type: keyword

example: CERT


**`santa.mode`**
:   Operating mode of Santa.

type: keyword

example: M



## disk [_disk]

Fields for DISKAPPEAR actions.

**`santa.disk.volume`**
:   The volume name.


**`santa.disk.bus`**
:   The disk bus protocol.


**`santa.disk.serial`**
:   The disk serial number.


**`santa.disk.bsdname`**
:   The disk BSD name.

example: disk1s3


**`santa.disk.model`**
:   The disk model.

example: APPLE SSD SM0512L


**`santa.disk.fs`**
:   The disk volume kind (filesystem type).

example: apfs


**`santa.disk.mount`**
:   The disk volume path.


**`santa.certificate.common_name`**
:   Common name from code signing certificate.

type: keyword


**`santa.certificate.sha256`**
:   SHA256 hash of code signing certificate.

type: keyword


