---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-jolokia-autodiscover.html
---

# Jolokia Discovery autodiscover provider fields [exported-fields-jolokia-autodiscover]

Metadata from Jolokia Discovery added by the jolokia provider.

**`jolokia.agent.version`**
:   Version number of jolokia agent.

type: keyword


**`jolokia.agent.id`**
:   Each agent has a unique id which can be either provided during startup of the agent in form of a configuration parameter or being autodetected. If autodected, the id has several parts: The IP, the process id, hashcode of the agent and its type.

type: keyword


**`jolokia.server.product`**
:   The container product if detected.

type: keyword


**`jolokia.server.version`**
:   The container’s version (if detected).

type: keyword


**`jolokia.server.vendor`**
:   The vendor of the container the agent is running in.

type: keyword


**`jolokia.url`**
:   The URL how this agent can be contacted.

type: keyword


**`jolokia.secured`**
:   Whether the agent was configured for authentication or not.

type: boolean


