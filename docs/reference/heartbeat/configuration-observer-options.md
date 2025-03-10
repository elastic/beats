---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/configuration-observer-options.html
---

# Add observer and geo metadata [configuration-observer-options]

Use the `heartbeat.run_from` option to set the geographic location fields relevant to a given heartbeat instance.

The `run_from` option is used to label the geographic location where the monitor is running. Note, you can also set the `run_from` option on an individual monitor to apply a unique setting to just that monitor.

The `run_from` option takes two top-level fields:

* `id`: A string used to uniquely identify the geographic location. It is indexed as the `observer.name` field.
* `geo`: A map conforming to [ECS geo fields](ecs://reference/ecs-geo.md). It is indexed under `observer.geo`.

Example:

```yaml
heartbeat.run_from:
  id: my-custom-geo
  geo:
	name: nyc-dc1-rack2
	location: 40.7128, -74.0060
	continent_name: North America
	country_iso_code: US
	region_name: New York
	region_iso_code: NY
	city_name: New York
```

