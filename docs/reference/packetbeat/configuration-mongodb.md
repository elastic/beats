---
navigation_title: "MongoDB"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-mongodb.html
---

# Capture MongoDB traffic [configuration-mongodb]


The following settings are specific to the MongoDB protocol. Here is a sample configuration for the `mongodb` section of the `packetbeat.yml` config file:

```yaml
packetbeat.protocols:
- type: mongodb
  send_request: true
  send_response: true
  max_docs: 0
  max_doc_length: 0
```

## Configuration options [_configuration_options_11]

The `max_docs` and `max_doc_length` settings are useful for limiting the amount of data Packetbeat indexes in the `response` fields.

Also see [Common protocol options](/reference/packetbeat/common-protocol-options.md).

### `max_docs` [_max_docs]

The maximum number of documents from the response to index in the `response` field. The default is 10. You can set this to 0 to index an unlimited number of documents.

Packetbeat adds a `[...]` line at the end to signify that there were additional documents that weren’t saved because of this setting.


### `max_doc_length` [_max_doc_length]

The maximum number of characters in a single document indexed in the `response` field. The default is 5000. You can set this to 0 to index an unlimited number of characters per document.

If the document is trimmed because of this setting, Packetbeat adds the string `...` at the end of the document.

Note that limiting documents in this way means that they are no longer correctly formatted JSON objects.



