---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-mysql.html
---

# MySQL fields [exported-fields-mysql]

MySQL-specific event fields.

**`mysql.affected_rows`**
:   If the MySQL command is successful, this field contains the affected number of rows of the last statement.

type: long


**`mysql.insert_id`**
:   If the INSERT query is successful, this field contains the id of the newly inserted row.


**`mysql.num_fields`**
:   If the SELECT query is successful, this field is set to the number of fields returned.


**`mysql.num_rows`**
:   If the SELECT query is successful, this field is set to the number of rows returned.


**`mysql.query`**
:   The row mysql query as read from the transaction’s request.


**`mysql.error_code`**
:   The error code returned by MySQL.

type: long


**`mysql.error_message`**
:   The error info message returned by MySQL.


