---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-cassandra.html
---

# Cassandra fields [exported-fields-cassandra]

Cassandra v4/3 specific event fields.

**`no_request`**
:   type: alias

alias to: cassandra.no_request



## cassandra [_cassandra]

Information about the Cassandra request and response.

**`cassandra.no_request`**
:   Indicates that there is no request because this is a PUSH message.

type: boolean



## request [_request]

Cassandra request.


## headers [_headers_3]

Cassandra request headers.

**`cassandra.request.headers.version`**
:   The version of the protocol.

type: long


**`cassandra.request.headers.flags`**
:   Flags applying to this frame.

type: keyword


**`cassandra.request.headers.stream`**
:   A frame has a stream id.  If a client sends a request message with the stream id X, it is guaranteed that the stream id of the response to that message will be X.

type: keyword


**`cassandra.request.headers.op`**
:   An operation type that distinguishes the actual message.

type: keyword


**`cassandra.request.headers.length`**
:   A integer representing the length of the body of the frame (a frame is limited to 256MB in length).

type: long


**`cassandra.request.query`**
:   The CQL query which client send to cassandra.

type: keyword



## response [_response]

Cassandra response.


## headers [_headers_4]

Cassandra response headers, the structure is as same as request’s header.

**`cassandra.response.headers.version`**
:   The version of the protocol.

type: long


**`cassandra.response.headers.flags`**
:   Flags applying to this frame.

type: keyword


**`cassandra.response.headers.stream`**
:   A frame has a stream id.  If a client sends a request message with the stream id X, it is guaranteed that the stream id of the response to that message will be X.

type: keyword


**`cassandra.response.headers.op`**
:   An operation type that distinguishes the actual message.

type: keyword


**`cassandra.response.headers.length`**
:   A integer representing the length of the body of the frame (a frame is limited to 256MB in length).

type: long



## result [_result]

Details about the returned result.

**`cassandra.response.result.type`**
:   Cassandra result type.

type: keyword



## rows [_rows]

Details about the rows.

**`cassandra.response.result.rows.num_rows`**
:   Representing the number of rows present in this result.

type: long



## meta [_meta]

Composed of result metadata.

**`cassandra.response.result.rows.meta.keyspace`**
:   Only present after set Global_tables_spec, the keyspace name.

type: keyword


**`cassandra.response.result.rows.meta.table`**
:   Only present after set Global_tables_spec, the table name.

type: keyword


**`cassandra.response.result.rows.meta.flags`**
:   Provides information on the formatting of the remaining information.

type: keyword


**`cassandra.response.result.rows.meta.col_count`**
:   Representing the number of columns selected by the query that produced this result.

type: long


**`cassandra.response.result.rows.meta.pkey_columns`**
:   Representing the PK columns index and counts.

type: long


**`cassandra.response.result.rows.meta.paging_state`**
:   The paging_state is a bytes value that should be used in QUERY/EXECUTE to continue paging and retrieve the remainder of the result for this query.

type: keyword


**`cassandra.response.result.keyspace`**
:   Indicating the name of the keyspace that has been set.

type: keyword



## schema_change [_schema_change]

The result to a schema_change message.

**`cassandra.response.result.schema_change.change`**
:   Representing the type of changed involved.

type: keyword


**`cassandra.response.result.schema_change.keyspace`**
:   This describes which keyspace has changed.

type: keyword


**`cassandra.response.result.schema_change.table`**
:   This describes which table has changed.

type: keyword


**`cassandra.response.result.schema_change.object`**
:   This describes the name of said affected object (either the table, user type, function, or aggregate name).

type: keyword


**`cassandra.response.result.schema_change.target`**
:   Target could be "FUNCTION" or "AGGREGATE", multiple arguments.

type: keyword


**`cassandra.response.result.schema_change.name`**
:   The function/aggregate name.

type: keyword


**`cassandra.response.result.schema_change.args`**
:   One string for each argument type (as CQL type).

type: keyword



## prepared [_prepared]

The result to a PREPARE message.

**`cassandra.response.result.prepared.prepared_id`**
:   Representing the prepared query ID.

type: keyword



## req_meta [_req_meta]

This describes the request metadata.

**`cassandra.response.result.prepared.req_meta.keyspace`**
:   Only present after set Global_tables_spec, the keyspace name.

type: keyword


**`cassandra.response.result.prepared.req_meta.table`**
:   Only present after set Global_tables_spec, the table name.

type: keyword


**`cassandra.response.result.prepared.req_meta.flags`**
:   Provides information on the formatting of the remaining information.

type: keyword


**`cassandra.response.result.prepared.req_meta.col_count`**
:   Representing the number of columns selected by the query that produced this result.

type: long


**`cassandra.response.result.prepared.req_meta.pkey_columns`**
:   Representing the PK columns index and counts.

type: long


**`cassandra.response.result.prepared.req_meta.paging_state`**
:   The paging_state is a bytes value that should be used in QUERY/EXECUTE to continue paging and retrieve the remainder of the result for this query.

type: keyword



## resp_meta [_resp_meta]

This describes the metadata for the result set.

**`cassandra.response.result.prepared.resp_meta.keyspace`**
:   Only present after set Global_tables_spec, the keyspace name.

type: keyword


**`cassandra.response.result.prepared.resp_meta.table`**
:   Only present after set Global_tables_spec, the table name.

type: keyword


**`cassandra.response.result.prepared.resp_meta.flags`**
:   Provides information on the formatting of the remaining information.

type: keyword


**`cassandra.response.result.prepared.resp_meta.col_count`**
:   Representing the number of columns selected by the query that produced this result.

type: long


**`cassandra.response.result.prepared.resp_meta.pkey_columns`**
:   Representing the PK columns index and counts.

type: long


**`cassandra.response.result.prepared.resp_meta.paging_state`**
:   The paging_state is a bytes value that should be used in QUERY/EXECUTE to continue paging and retrieve the remainder of the result for this query.

type: keyword


**`cassandra.response.supported`**
:   Indicates which startup options are supported by the server. This message comes as a response to an OPTIONS message.

type: object



## authentication [_authentication]

Indicates that the server requires authentication, and which authentication mechanism to use.

**`cassandra.response.authentication.class`**
:   Indicates the full class name of the IAuthenticator in use

type: keyword


**`cassandra.response.warnings`**
:   The text of the warnings, only occur when Warning flag was set.

type: keyword



## event [_event]

Event pushed by the server. A client will only receive events for the types it has REGISTERed to.

**`cassandra.response.event.type`**
:   Representing the event type.

type: keyword


**`cassandra.response.event.change`**
:   The message corresponding respectively to the type of change followed by the address of the new/removed node.

type: keyword


**`cassandra.response.event.host`**
:   Representing the node ip.

type: keyword


**`cassandra.response.event.port`**
:   Representing the node port.

type: long



## schema_change [_schema_change_2]

The events details related to schema change.

**`cassandra.response.event.schema_change.change`**
:   Representing the type of changed involved.

type: keyword


**`cassandra.response.event.schema_change.keyspace`**
:   This describes which keyspace has changed.

type: keyword


**`cassandra.response.event.schema_change.table`**
:   This describes which table has changed.

type: keyword


**`cassandra.response.event.schema_change.object`**
:   This describes the name of said affected object (either the table, user type, function, or aggregate name).

type: keyword


**`cassandra.response.event.schema_change.target`**
:   Target could be "FUNCTION" or "AGGREGATE", multiple arguments.

type: keyword


**`cassandra.response.event.schema_change.name`**
:   The function/aggregate name.

type: keyword


**`cassandra.response.event.schema_change.args`**
:   One string for each argument type (as CQL type).

type: keyword



## error [_error]

Indicates an error processing a request. The body of the message will be an  error code followed by a error message. Then, depending on the exception, more content may follow.

**`cassandra.response.error.code`**
:   The error code of the Cassandra response.

type: long


**`cassandra.response.error.msg`**
:   The error message of the Cassandra response.

type: keyword


**`cassandra.response.error.type`**
:   The error type of the Cassandra response.

type: keyword



## details [_details]

The details of the error.

**`cassandra.response.error.details.read_consistency`**
:   Representing the consistency level of the query that triggered the exception.

type: keyword


**`cassandra.response.error.details.required`**
:   Representing the number of nodes that should be alive to respect consistency level.

type: long


**`cassandra.response.error.details.alive`**
:   Representing the number of replicas that were known to be alive when the request had been processed (since an unavailable exception has been triggered).

type: long


**`cassandra.response.error.details.received`**
:   Representing the number of nodes having acknowledged the request.

type: long


**`cassandra.response.error.details.blockfor`**
:   Representing the number of replicas whose acknowledgement is required to achieve consistency level.

type: long


**`cassandra.response.error.details.write_type`**
:   Describe the type of the write that timed out.

type: keyword


**`cassandra.response.error.details.data_present`**
:   It means the replica that was asked for data had responded.

type: boolean


**`cassandra.response.error.details.keyspace`**
:   The keyspace of the failed function.

type: keyword


**`cassandra.response.error.details.table`**
:   The keyspace of the failed function.

type: keyword


**`cassandra.response.error.details.stmt_id`**
:   Representing the unknown ID.

type: keyword


**`cassandra.response.error.details.num_failures`**
:   Representing the number of nodes that experience a failure while executing the request.

type: keyword


**`cassandra.response.error.details.function`**
:   The name of the failed function.

type: keyword


**`cassandra.response.error.details.arg_types`**
:   One string for each argument type (as CQL type) of the failed function.

type: keyword


