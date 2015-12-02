# Mongodb protocol parsing for packetbeat

Main documentation link:

  - [legacy documentation of the wire protocol](http://docs.mongodb.org/meta-driver/latest/legacy/mongodb-wire-protocol/)
  - [documentation of all database commands](http://docs.mongodb.org/manual/reference/command/)

## Understanding wire protocol vs command

At first it is difficult to understand how the legacy protocol fits with the concept of 'command' which is always repeated in the doc but not very well explained (or not where I looked).

This [mail thread](https://groups.google.com/forum/#!topic/mongodb-dev/3k2YGJYRZms) fortunately gave the answer: "GetLastError is a command and command are implemented using findOne, which generates an OP_QUERY message."

In the write operations as commands mode which seems to be the current mode, the response is therefore a 'OP_REPLY' message and there will always be one to close the transaction.

In the case of write operations as separate message types, we should parse the following 'getLastError' command and consider it as part of the same transaction, the response to this command actually being the response to the original write operation. Except that the getLastError command is optional, the client will not send it if it was requested with a write concern of 0. This mode is only supported by clients dans database as a legacy mode, it will be supported by this parser only very basically.

## TODO

  - Support option to send documents in response (Send_Response ?)
  - Support option to send update and insert documents in request (Send_Request ?)
  - Support option to ignore non user commands
  - Fill bytes_in and bytes_out
