---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-postgresql.html
---

# PostgreSQL fields [exported-fields-postgresql]

Module for parsing the PostgreSQL log files.


## postgresql [_postgresql]

Fields from PostgreSQL logs.


## log [_log_11]

Fields from the PostgreSQL log files.

**`postgresql.log.timestamp`**
:   :::{admonition} Deprecated in 7.3.0
    The `postgresql.log.timestamp` field was deprecated in 7.3.0.
    :::

The timestamp from the log line.


**`postgresql.log.core_id`**
:   :::{admonition} Deprecated in 8.0.0
    The `postgresql.log.core_id` field was deprecated in 8.0.0.
    :::

Core id. (deprecated, there is no core_id in PostgreSQL logs, this is actually session_line_number).

type: alias

alias to: postgresql.log.session_line_number


**`postgresql.log.client_addr`**
:   Host where the connection originated from.

example: 127.0.0.1


**`postgresql.log.client_port`**
:   Port where the connection originated from.

example: 59700


**`postgresql.log.session_id`**
:   PostgreSQL session.

example: 5ff1dd98.22


**`postgresql.log.session_line_number`**
:   Line number inside a session. (%l in `log_line_prefix`).

type: long


**`postgresql.log.database`**
:   Name of database.

example: postgres


**`postgresql.log.query`**
:   Query statement. In the case of CSV parse, look at command_tag to get more context.

example: SELECT * FROM users;


**`postgresql.log.query_step`**
:   Statement step when using extended query protocol (one of statement, parse, bind or execute).

example: parse


**`postgresql.log.query_name`**
:   Name given to a query when using extended query protocol. If it is "<unnamed>", or not present, this field is ignored.

example: pdo_stmt_00000001


**`postgresql.log.command_tag`**
:   Type of session’s current command. The complete list can be found at: src/include/tcop/cmdtaglist.h

example: SELECT


**`postgresql.log.session_start_time`**
:   Time when this session started.

type: date


**`postgresql.log.virtual_transaction_id`**
:   Backend local transaction id.


**`postgresql.log.transaction_id`**
:   The id of current transaction.

type: long


**`postgresql.log.sql_state_code`**
:   State code returned by Postgres (if any). See also [https://www.postgresql.org/docs/current/errcodes-appendix.html](https://www.postgresql.org/docs/current/errcodes-appendix.html)

type: keyword


**`postgresql.log.detail`**
:   More information about the message, parameters in case of a parametrized query. e.g. *Role \"user\" does not exist.*, *parameters: $1 = 42*, etc.


**`postgresql.log.hint`**
:   A possible solution to solve an error.


**`postgresql.log.internal_query`**
:   Internal query that led to the error (if any).


**`postgresql.log.internal_query_pos`**
:   Character count of the internal query (if any).

type: long


**`postgresql.log.context`**
:   Error context.


**`postgresql.log.query_pos`**
:   Character count of the error position (if any).

type: long


**`postgresql.log.location`**
:   Location of the error in the PostgreSQL source code (if log_error_verbosity is set to verbose).


**`postgresql.log.application_name`**
:   Name of the application of this event. It is defined by the client.


**`postgresql.log.backend_type`**
:   Type of backend of this event. Possible types are autovacuum launcher, autovacuum worker, logical replication launcher, logical replication worker, parallel worker, background writer, client backend, checkpointer, startup, walreceiver, walsender and walwriter. In addition, background workers registered by extensions may have additional types.

example: client backend


**`postgresql.log.error.code`**
:   :::{admonition} Deprecated in 8.0.0
    The `postgresql.log.error.code` field was deprecated in 8.0.0.
    :::

Error code returned by Postgres (if any). Deprecated: errors can have letters. Use sql_state_code instead.

type: alias

alias to: postgresql.log.sql_state_code


**`postgresql.log.timezone`**
:   type: alias

alias to: event.timezone


**`postgresql.log.user`**
:   type: alias

alias to: user.name


**`postgresql.log.level`**
:   Valid values are DEBUG5, DEBUG4, DEBUG3, DEBUG2, DEBUG1, INFO, NOTICE, WARNING, ERROR, LOG, FATAL, and PANIC.

type: alias

example: LOG

alias to: log.level


**`postgresql.log.message`**
:   type: alias

alias to: message


