---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-trans_event.html
---

# Transaction Event fields [exported-fields-trans_event]

These fields contain data about the transaction itself.

**`status`**
:   The high level status of the transaction. The way to compute this value depends on the protocol, but the result has a meaning independent of the protocol.

required: True


**`method`**
:   The command/verb/method of the transaction. For HTTP, this is the method name (GET, POST, PUT, and so on), for SQL this is the verb (SELECT, UPDATE, DELETE, and so on).


**`resource`**
:   The logical resource that this transaction refers to. For HTTP, this is the URL path up to the last slash (/). For example, if the URL is `/users/1`, the resource is `/users`. For databases, the resource is typically the table name. The field is not filled for all transaction types.


**`path`**
:   The path the transaction refers to. For HTTP, this is the URL. For SQL databases, this is the table name. For key-value stores, this is the key.

required: True


**`query`**
:   The query in a human readable format. For HTTP, it will typically be something like `GET /users/_search?name=test`. For MySQL, it is something like `SELECT id from users where name=test`.

type: keyword


**`params`**
:   The request parameters. For HTTP, these are the POST or GET parameters. For Thrift-RPC, these are the parameters from the request.

type: text


**`notes`**
:   Messages from Packetbeat itself. This field usually contains error messages for interpreting the raw data. This information can be helpful for troubleshooting.

type: alias

alias to: error.message


