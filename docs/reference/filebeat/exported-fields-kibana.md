---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-kibana.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Kibana fields [exported-fields-kibana]

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

    example: ['default', 'marketing']


**`kibana.delete_from_spaces`**
:   The set of space ids that a saved object was removed from.

    type: keyword

    example: ['default', 'marketing']


**`kibana.diff.format`**
:   The schema of the saved object attribute diff attached to this event.

    type: keyword

    example: json_patch_extended


## diff.ops [_diff.ops]

JSON Patch operations describing the saved object attributes changed by this event, emitted when saved object diff auditing is enabled in Kibana. Each operation may include `value` and `oldValue` whose JSON type varies by attribute (string, number, boolean, array, or object). Those fields are stored in `_source` but not mapped (`dynamic: false` on this object) so mixed types cannot cause mapping conflicts. Mapping them as `object` with `enabled: false` is not sufficient: Elasticsearch still rejects non-object values such as strings.

**`kibana.diff.ops.op`**
:   The operation performed on the attribute.

    type: keyword

    example: replace


**`kibana.diff.ops.path`**
:   RFC 6901 JSON Pointer to the changed attribute.

    type: keyword

    example: /title


**`kibana.diff.noOps.path`**
:   RFC 6901 JSON Pointer to an attribute that was not changed by this event.

    type: keyword

    example: /description


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


## log [_log]

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


