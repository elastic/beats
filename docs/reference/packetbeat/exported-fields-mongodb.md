---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-mongodb.html
---

# MongoDb fields [exported-fields-mongodb]

MongoDB-specific event fields. These fields mirror closely the fields for the MongoDB wire protocol. The higher level fields (for example, `query` and `resource`) apply to MongoDB events as well.

**`mongodb.error`**
:   If the MongoDB request has resulted in an error, this field contains the error message returned by the server.


**`mongodb.fullCollectionName`**
:   The full collection name. The full collection name is the concatenation of the database name with the collection name, using a dot (.) for the concatenation. For example, for the database foo and the collection bar, the full collection name is foo.bar.


**`mongodb.numberToSkip`**
:   Sets the number of documents to omit - starting from the first document in the resulting dataset - when returning the result of the query.

type: long


**`mongodb.numberToReturn`**
:   The requested maximum number of documents to be returned.

type: long


**`mongodb.numberReturned`**
:   The number of documents in the reply.

type: long


**`mongodb.startingFrom`**
:   Where in the cursor this reply is starting.


**`mongodb.query`**
:   A JSON document that represents the query. The query will contain one or more elements, all of which must match for a document to be included in the result set. Possible elements include $query, $orderby, $hint, $explain, and $snapshot.


**`mongodb.returnFieldsSelector`**
:   A JSON document that limits the fields in the returned documents. The returnFieldsSelector contains one or more elements, each of which is the name of a field that should be returned, and the integer value 1.


**`mongodb.selector`**
:   A BSON document that specifies the query for selecting the document to update or delete.


**`mongodb.update`**
:   A BSON document that specifies the update to be performed. For information on specifying updates, see the Update Operations documentation from the MongoDB Manual.


**`mongodb.cursorId`**
:   The cursor identifier returned in the OP_REPLY. This must be the value that was returned from the database.


