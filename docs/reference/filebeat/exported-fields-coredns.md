---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-coredns.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# CoreDNS fields [exported-fields-coredns]

Module for handling logs produced by coredns

## coredns [_coredns]

coredns fields after normalization

**`coredns.query.size`**
:   size of the DNS query

    type: integer

    format: bytes


**`coredns.response.size`**
:   size of the DNS response

    type: integer

    format: bytes


