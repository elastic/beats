---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/event-conventions.html
---

# Naming Conventions [event-conventions]

When creating events, use the following conventions for field names and abbreviations.

## Field Names [field-names]

Use the following naming conventions for field names:

* All fields must be lower case.
* Use snake case (underscores) for combining words.
* Group related fields into subdocuments by using dot (.) notation. Groups typically have common prefixes. For example, if you have fields called `CPULoad` and `CPUSystem` in a service, you would convert them into `cpu.load` and `cpu.system` in the event.
* Avoid repeating the namespace in field names. If a word or abbreviation appears in the namespace, it’s not needed in the field name. For example, instead of `cpu.cpu_load`, use `cpu.load`.
* Use [units suffix](#units) when the metric matches one of the known units.
* Use [standardised names](#abbreviations) and avoid using abbreviations that aren’t commonly known.
* Organise the documents from general to specific to allow for namespacing. The type, such as `.pct`, should always be last. For example, `system.core.user.pct`.
* If two fields are the same, but with different units, remove the less granular one. For example, include `timeout.sec`, but don’t include `timeout.min`. If a less granular value is required, you can calculate it later.
* If a field name matches the namespace used for nested fields, add `.value` to the field name. For example, instead of:

    ```yaml
    workers
    workers.busy
    workers.idle
    ```

    Use:

    ```yaml
    workers.value
    workers.busy
    workers.idle
    ```

* Do not use dots (.) in individual field names. Dots are reserved for grouping related fields into subdocuments.
* Use singular and plural names properly to reflect the field content. For example, use `requests_per_sec` rather than `request_per_sec`.


## Units [units]

These are well-known suffixes to represent units of stored values, use them as a dotted suffix when possible. For example `system.memory.used.bytes` or `system.diskio.read.count`:

| Suffix | Units |
| --- | --- |
| count | item count |
| pct | percentage |
| day | days |
| sec | seconds |
| ms | millisecond |
| us | microseconds |
| ns | nanoseconds |
| bytes | bytes |
| mb | megabytes |


## Standardised Names [abbreviations]

Here is a list of standardised names and units that are used across all Beats:

| Use…​ | Instead of…​ |
| --- | --- |
| avg | average |
| connection | conn |
| max | maximum |
| min | minimum |
| request | req |
| msg | message |


