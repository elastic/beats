---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-elasticsearch-querylog.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Elasticsearch query log fields [exported-fields-elasticsearch-querylog]

Root-level fields from Elasticsearch query log JSON lines when ingested with the filestream NDJSON parser and expand_keys.

**`http.request.headers.x_opaque_id`**
:   Value of the X-Opaque-Id HTTP header when nested under http.request.headers in ECS-style logs.

    type: keyword


**`user.realm`**
:   Authentication realm for the user in Elasticsearch structured logging.

    type: keyword


**`auth.type`**
:   Authentication mechanism (TOKEN, REALM, API_KEY, etc.) from Elasticsearch structured logging.

    type: keyword


