---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-kibana.html
---

# kibana fields [exported-fields-kibana]

kibana Module

**`service.node.roles`**
:   type: keyword



## kibana [_kibana]

Module for parsing Kibana logs.

**`kibana.session_id`**
:   The ID of the user session associated with this event. Each login attempt results in a unique session id.

type: keyword

example: 123e4567-e89b-12d3-a456-426614174000


**`kibana.space_id`**
:   The id of the space associated with this event.

type: keyword

example: default


**`kibana.saved_object.type`**
:   The type of the saved object associated with this event.

type: keyword

example: dashboard


**`kibana.saved_object.id`**
:   The id of the saved object associated with this event.

type: keyword

example: 6295bdd0-0a0e-11e7-825f-6748cda7d858


**`kibana.saved_object.name`**
:   The name of the saved object associated with this event.

type: keyword

example: my-saved-object


**`kibana.add_to_spaces`**
:   The set of space ids that a saved object was shared to.

type: keyword

example: [*default*, *marketing*]


**`kibana.delete_from_spaces`**
:   The set of space ids that a saved object was removed from.

type: keyword

example: [*default*, *marketing*]


**`kibana.authentication_provider`**
:   The authentication provider associated with a login event.

type: keyword

example: basic1


**`kibana.authentication_type`**
:   The authentication provider type associated with a login event.

type: keyword

example: basic


**`kibana.authentication_realm`**
:   The Elasticsearch authentication realm name which fulfilled a login event.

type: keyword

example: native


**`kibana.lookup_realm`**
:   The Elasticsearch lookup realm which fulfilled a login event.

type: keyword

example: native



## log [_log_6]

Kibana log lines.

**`kibana.log.tags`**
:   Kibana logging tags.

type: keyword


**`kibana.log.state`**
:   Current state of Kibana.

type: keyword


**`kibana.log.meta`**
:   type: object


**`kibana.log.meta.req.headers`**
:   type: flattened


**`kibana.log.meta.res.headers`**
:   type: flattened


