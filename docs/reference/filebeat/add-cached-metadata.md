---
navigation_title: "cache"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/add-cached-metadata.html
applies_to:
  stack: preview
---

# Add cached metadata [add-cached-metadata]


::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


The `cache` processor enriches events with information from a previously cached events.

```yaml
processors:
  - cache:
      backend:
        memory:
          id: cache_id
      put:
        key_field: join_key_field
        value_field: source_field
```

```yaml
processors:
  - cache:
      backend:
        memory:
          id: cache_id
      get:
        key_field: join_key_field
        target_field: destination_field
```

```yaml
processors:
  - cache:
      backend:
        memory:
          id: cache_id
      delete:
        key_field: join_key_field
```

The fields added to the target field will depend on the provider.

It has the following settings:

One of `backend.memory.id` or `backend.file.id` must be provided.

`backend.capacity`
:   The number of elements that can be stored in the cache. `put` operations that would cause the capacity to be exceeded will result in evictions of the oldest elements. Values at or below zero indicate no limit. The capacity should not be lower than the number of elements that are expected to be referenced when processing the input as evicted elements are lost. The default is `0`, no limit.

`backend.memory.id`
:   The ID of a memory-based cache. Use the same ID across instance to reference the same cache.

`backend.file.id`
:   The ID of a file-based cache. Use the same ID across instance to reference the same cache.

`backend.file.write_interval`
:   The interval between periodic cache writes to the backing file. Valid time units are h, m, s, ms, us/µs and ns. Periodic writes are only made if `backend.file.write_interval` is greater than zero. The contents are always written out to the backing file when the processor is closed. Default is zero, no periodic writes.

One of `put`, `get` or `delete` must be provided.

`put.key_field`
:   Name of the field containing the key to put into the cache. Required if `put` is present.

`put.value_field`
:   Name of the field containing the value to put into the cache. Required if `put` is present.

`put.ttl`
:   The TTL to associate with the cached key/value. Valid time units are h, m, s, ms, us/µs and ns. Required if `put` is present.

`get.key_field`
:   Name of the field containing the key to get. Required if `get` is present.

`get.target_field`
:   Name of the field to which the cached value will be written. Required if `get` is present.

`delete.key_field`
:   Name of the field containing the key to delete. Required if `delete` is present.

`ignore_missing`
:   (Optional) When set to `false`, events that don’t contain any of the fields in `match_keys` will be discarded and an error will be generated. By default, this condition is ignored.

`overwrite_keys`
:   (Optional) By default, if a target field already exists, it will not be overwritten and an error will be logged. If `overwrite_keys` is set to `true`, this condition will be ignored.

The `cache` processor can be used to perform joins within the Beat between documents within an event stream.

```yaml
processors:
  - if:
      contains:
        log.file.path: fdrv2/aidmaster
    then:
      - cache:
          backend:
            memory:
              id: aidmaster
            capacity: 10000
          put:
            ttl: 168h
            key_field: crowdstrike.aid
            value_field: crowdstrike.metadata
    else:
      - cache:
          backend:
            memory:
              id: aidmaster
          get:
            key_field: crowdstrike.aid
            target_field: crowdstrike.metadata
```

This would enrich an event events with `log.file.path` not equal to "fdrv2/aidmaster" with the `crowdstrike.metadata` fields from events with `log.file.path` equal to that value where the `crowdstrike.aid` field matches between the source and destination documents. The capacity allows up to 10,000 metadata object to be cached between `put` and `get` operations.

