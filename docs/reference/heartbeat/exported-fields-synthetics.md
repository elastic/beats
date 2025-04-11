---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-synthetics.html
---

# Synthetics types fields [exported-fields-synthetics]

None


## synthetics [_synthetics]

Synthetics related fields.

**`synthetics.type`**
:   type: keyword


**`synthetics.package_version`**
:   type: keyword


**`synthetics.index`**
:   Index count used for creating total order of all events during invocation.

type: integer


**`synthetics.payload`**
:   type: object

Object is not enabled.


**`synthetics.blob`**
:   binary data payload

type: binary


**`synthetics.blob_mime`**
:   mime type of blob data

type: keyword


**`synthetics.step.name`**
:   type: text


**`synthetics.step.name.keyword`**
:   type: keyword


**`synthetics.step.index`**
:   type: integer


**`synthetics.step.status`**
:   type: keyword



## duration [_duration_3]

Duration required to complete the step.

**`synthetics.step.duration.us`**
:   Duration in microseconds

type: integer


**`synthetics.journey.name`**
:   type: text


**`synthetics.journey.id`**
:   type: keyword


**`synthetics.journey.tags`**
:   Tags used for grouping journeys

type: keyword



## duration [_duration_4]

Duration required to complete the journey.

**`synthetics.journey.duration.us`**
:   Duration in microseconds

type: integer


**`synthetics.error.name`**
:   type: keyword


**`synthetics.error.message`**
:   type: text


**`synthetics.error.stack`**
:   type: text


**`synthetics.screenshot_ref.width`**
:   Width of the full screenshot in pixels.

type: integer


**`synthetics.screenshot_ref.height`**
:   Height of the full screenshot in pixels

type: integer



## blocks [_blocks]

Attributes representing individual screenshot blocks. Only hash is indexed since it’s the only one we’d query on.

**`synthetics.screenshot_ref.blocks.hash`**
:   Hash that uniquely identifies this image by content. Corresponds to block document id.

type: keyword


