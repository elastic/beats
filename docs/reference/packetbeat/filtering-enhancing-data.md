---
navigation_title: "Processors"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/filtering-and-enhancing-data.html
---

# Filter and enhance data with processors [filtering-and-enhancing-data]


You can [define processors](/reference/packetbeat/defining-processors.md) in your configuration to process events before they are sent to the configured output. The libbeat library provides processors for:

* reducing the number of exported fields
* enhancing events with additional metadata
* performing additional processing and decoding

Each processor receives an event, applies a defined action to the event, and returns the event. If you define a list of processors, they are executed in the order they are defined in the Packetbeat configuration file.

```yaml
event -> processor 1 -> event1 -> processor 2 -> event2 ...
```

::::{important}
It’s recommended to do all drop and renaming of existing fields as the last step in a processor configuration. This is because dropping or renaming fields can remove data necessary for the next processor in the chain, for example dropping the `source.ip` field would remove one of the fields necessary for the `community_id` processor to function. If it’s necessary to remove, rename or overwrite an existing event field, please make sure it’s done by a corresponding processor ([`drop_fields`](/reference/packetbeat/drop-fields.md), [`rename`](/reference/packetbeat/rename-fields.md) or [`add_fields`](/reference/packetbeat/add-fields.md)) placed at the end of the processor list defined in the input configuration.
::::


For example, the following configuration includes a subset of the Packetbeat DNS fields so that only the requests and their response codes are reported:

```yaml
processors:
  - include_fields:
      fields:
        - client.bytes
        - server.bytes
        - client.ip
        - server.ip
        - dns.question.name
        - dns.question.etld_plus_one
        - dns.response_code
```

The filtered event would look something like this:

```shell
{
  "@timestamp": "2019-01-19T03:41:11.798Z",
  "client": {
    "bytes": 28,
    "ip": "10.100.6.82"
  },
  "server": {
    "bytes": 271,
    "ip": "10.100.4.1"
  },
  "dns": {
    "question": {
      "name": "www.elastic.co",
      "etld_plus_one": "elastic.co"
    },
    "response_code": "NOERROR"
  },
  "type": "dns"
}
```

If you would like to drop all the successful transactions, you can use the following configuration:

```yaml
processors:
  - drop_event:
      when:
        equals:
          http.response.status_code: 200
```

If you don’t want to export raw data for the successful transactions:

```yaml
processors:
  - drop_fields:
      when:
        equals:
          http.response.status_code: 200
      fields: ["request", "response"]
```












































