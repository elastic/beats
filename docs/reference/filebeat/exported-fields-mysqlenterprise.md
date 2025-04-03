---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-mysqlenterprise.html
---

# MySQL Enterprise fields [exported-fields-mysqlenterprise]

MySQL Enterprise Audit module


## mysqlenterprise [_mysqlenterprise]

Fields from MySQL Enterprise Logs


## audit [_audit_4]

Module for parsing MySQL Enterprise Audit Logs

**`mysqlenterprise.audit.class`**
:   A string representing the event class. The class defines the type of event, when taken together with the event item that specifies the event subclass.

type: keyword


**`mysqlenterprise.audit.connection_id`**
:   An integer representing the client connection identifier. This is the same as the value returned by the CONNECTION_ID() function within the session.

type: keyword


**`mysqlenterprise.audit.id`**
:   An unsigned integer representing an event ID.

type: keyword


**`mysqlenterprise.audit.connection_data.connection_type`**
:   The security state of the connection to the server. Permitted values are tcp/ip (TCP/IP connection established without encryption), ssl (TCP/IP connection established with encryption), socket (Unix socket file connection), named_pipe (Windows named pipe connection), and shared_memory (Windows shared memory connection).

type: keyword


**`mysqlenterprise.audit.connection_data.status`**
:   An integer representing the command status: 0 for success, nonzero if an error occurred.

type: long


**`mysqlenterprise.audit.connection_data.db`**
:   A string representing a database name. For connection_data, it is the default database. For table_access_data, it is the table database.

type: keyword


**`mysqlenterprise.audit.connection_data.connection_attributes`**
:   Connection attributes that might be passed by different MySQL Clients.

type: flattened


**`mysqlenterprise.audit.general_data.command`**
:   A string representing the type of instruction that generated the audit event, such as a command that the server received from a client.

type: keyword


**`mysqlenterprise.audit.general_data.sql_command`**
:   A string that indicates the SQL statement type.

type: keyword


**`mysqlenterprise.audit.general_data.query`**
:   A string representing the text of an SQL statement. The value can be empty. Long values may be truncated. The string, like the audit log file itself, is written using UTF-8 (up to 4 bytes per character), so the value may be the result of conversion.

type: keyword


**`mysqlenterprise.audit.general_data.status`**
:   An integer representing the command status: 0 for success, nonzero if an error occurred. This is the same as the value of the mysql_errno() C API function.

type: long


**`mysqlenterprise.audit.login.user`**
:   A string representing the information indicating how a client connected to the server.

type: keyword


**`mysqlenterprise.audit.login.proxy`**
:   A string representing the proxy user. The value is empty if user proxying is not in effect.

type: keyword


**`mysqlenterprise.audit.shutdown_data.server_id`**
:   An integer representing the server ID. This is the same as the value of the server_id system variable.

type: keyword


**`mysqlenterprise.audit.startup_data.server_id`**
:   An integer representing the server ID. This is the same as the value of the server_id system variable.

type: keyword


**`mysqlenterprise.audit.startup_data.mysql_version`**
:   An integer representing the server ID. This is the same as the value of the server_id system variable.

type: keyword


**`mysqlenterprise.audit.table_access_data.db`**
:   A string representing a database name. For connection_data, it is the default database. For table_access_data, it is the table database.

type: keyword


**`mysqlenterprise.audit.table_access_data.table`**
:   A string representing a table name.

type: keyword


**`mysqlenterprise.audit.table_access_data.query`**
:   A string representing the text of an SQL statement. The value can be empty. Long values may be truncated. The string, like the audit log file itself, is written using UTF-8 (up to 4 bytes per character), so the value may be the result of conversion.

type: keyword


**`mysqlenterprise.audit.table_access_data.sql_command`**
:   A string that indicates the SQL statement type.

type: keyword


**`mysqlenterprise.audit.account.user`**
:   A string representing the user that the server authenticated the client as. This is the user name that the server uses for privilege checking.

type: keyword


**`mysqlenterprise.audit.account.host`**
:   A string representing the client host name.

type: keyword


**`mysqlenterprise.audit.login.os`**
:   A string representing the external user name used during the authentication process, as set by the plugin used to authenticate the client.

type: keyword


