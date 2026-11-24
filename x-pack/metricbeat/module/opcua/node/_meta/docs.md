The `node` metricset reads the value of the configured OPC UA nodes in one read request per period and creates one event per node.

Each event holds the node id, the configured label, the display name read from the server, the data type, the status code with its severity, and the source and server timestamps. The value is stored in one of the typed fields `value.double`, `value.long`, `value.boolean`, `value.keyword` or `value.date`, so that each field keeps a single type across nodes. Array values and structured values, such as ByteString or ExtensionObject, are not indexed; their data type is still reported.

A node that can't be read, for example because its id is unknown, produces an event with a bad status and no value.
