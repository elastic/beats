---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-apache.html
---

# Apache fields [exported-fields-apache]

Apache Module


## apache [_apache]

Apache fields.


## access [_access]

Contains fields for the Apache HTTP Server access logs.

**`apache.access.ssl.protocol`**
:   SSL protocol version.

type: keyword


**`apache.access.ssl.cipher`**
:   SSL cipher name.

type: keyword



## error [_error]

Fields from the Apache error logs.

**`apache.error.module`**
:   The module producing the logged message.

type: keyword


