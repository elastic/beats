---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-common.html
---

# Common fields [exported-fields-common]

These fields contain data about the environment in which the transaction or flow was captured.

**`type`**
:   The type of the transaction (for example, HTTP, MySQL, Redis, or RUM) or "flow" in case of flows.

required: True


**`server.process.name`**
:   The name of the process that served the transaction.


**`server.process.args`**
:   The command-line of the process that served the transaction.


**`server.process.executable`**
:   Absolute path to the server process executable.


**`server.process.working_directory`**
:   The working directory of the server process.


**`server.process.start`**
:   The time the server process started.


**`client.process.name`**
:   The name of the process that initiated the transaction.


**`client.process.args`**
:   The command-line of the process that initiated the transaction.


**`client.process.executable`**
:   Absolute path to the client process executable.


**`client.process.working_directory`**
:   The working directory of the client process.


**`client.process.start`**
:   The time the client process started.


**`real_ip`**
:   If the server initiating the transaction is a proxy, this field contains the original client IP address. For HTTP, for example, the IP address extracted from a configurable HTTP header, by default `X-Forwarded-For`. Unless this field is disabled, it always has a value, and it matches the `client_ip` for non proxy clients.

type: alias

alias to: network.forwarded_ip


**`transport`**
:   The transport protocol used for the transaction. If not specified, then tcp is assumed.

type: alias

alias to: network.transport


