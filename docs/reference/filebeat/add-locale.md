---
navigation_title: "add_locale"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/add-locale.html
---

# Add the local time zone [add-locale]


The `add_locale` processor enriches each event with the machine’s time zone offset from UTC or with the name of the time zone. It supports one configuration option named `format` that controls whether an offset or time zone abbreviation is added to the event. The default format is `offset`. The processor adds the a `event.timezone` value to each event.

The configuration below enables the processor with the default settings.

```yaml
processors:
  - add_locale: ~
```

This configuration enables the processor and configures it to add the time zone abbreviation to events.

```yaml
processors:
  - add_locale:
      format: abbreviation
```

::::{note}
Please note that `add_locale` differentiates between daylight savings time (DST) and regular time. For example `CEST` indicates DST and and `CET` is regular time.
::::


