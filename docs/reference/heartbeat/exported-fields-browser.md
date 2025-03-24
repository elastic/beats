---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-browser.html
---

# Synthetics browser metrics fields [exported-fields-browser]

None


## browser [_browser]

Browser metrics and traces


## experience [_experience]

Absolute values of all user experience metrics in the browser relative to the navigation start event in microseconds


## fcp [_fcp]

duration of First contentful paint metric

**`browser.experience.fcp.us`**
:   type: integer



## lcp [_lcp]

duration of Largest contentful paint metric

**`browser.experience.lcp.us`**
:   type: integer



## dcl [_dcl]

duration of Document content loaded end event

**`browser.experience.dcl.us`**
:   type: integer



## load [_load]

duration of Load end event

**`browser.experience.load.duration`**
:   type: integer


**`browser.experience.cls`**
:   culumative layout shift score across all frames

type: integer



## relative_trace [_relative_trace]

trace event with timing information that are realtive to journey timings in microseconds

**`browser.relative_trace.name`**
:   name of the trace event

type: keyword


**`browser.relative_trace.type`**
:   could be one of mark or measure event types

type: text



## start [_start]

monotonically increasing trace start time in microseconds

**`browser.relative_trace.start.us`**
:   type: long



## duration [_duration]

duration of the trace event in microseconds.

**`browser.relative_trace.duration.us`**
:   type: integer


**`browser.relative_trace.score`**
:   weighted score of the layout shift event

type: integer


