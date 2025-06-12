---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kvm.html
---

<!-- This file is generated! See scripts/generate_fields_docs.py -->

# KVM fields [exported-fields-kvm]

kvm module

**`kvm.id`**
:   Domain id

type: long


**`kvm.name`**
:   Domain name

type: keyword


## kvm [_kvm]



## dommemstat [_dommemstat]

dommemstat

## stat [_stat]

Memory stat

**`kvm.dommemstat.stat.name`**
:   Memory stat name

type: keyword


**`kvm.dommemstat.stat.value`**
:   Memory stat value

type: long


**`kvm.dommemstat.id`**
:   Domain id

type: long


**`kvm.dommemstat.name`**
:   Domain name

type: keyword


## status [_status]

status

**`kvm.status.state`**
:   Domain state

type: keyword


