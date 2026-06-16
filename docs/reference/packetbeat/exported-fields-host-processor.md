---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-host-processor.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Host fields [exported-fields-host-processor]

Info collected for the host machine.

**`host.containerized`**
:   If the host is a container.

    type: boolean


**`host.os.build`**
:   OS build information.

    type: keyword

    example: 18D109


**`host.os.codename`**
:   OS codename, if any.

    type: keyword

    example: stretch


