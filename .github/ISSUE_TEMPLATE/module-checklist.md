---
name: New Module / Dataset
about: "Meta issue to track the creation, updating of a new module or dataset."

---

# Metricbeat Module / Dataset release checklist

This checklist is intended for Devs which create or update a module to make sure modules are consistent.

## Modules

For a metricset to go GA, the following criterias should be met:

* [ ] Supported versions are documented
* [ ] Supported operating systems are documented (if applicable)
* [ ] Integration tests exist
* [ ] System tests exist
* [ ] Automated checks that all fields are documented
* [ ] Documentation
* [ ] Fields follow [ECS](https://github.com/elastic/ecs) and [naming conventions](https://www.elastic.co/guide/en/beats/devguide/master/event-conventions.html)
* [ ] Dashboards exists (if applicable)
* [ ] Kibana Home Tutorial (if applicable)
  * Open PR against Kibana repo with tutorial. Examples can be found [here](https://github.com/elastic/kibana/tree/master/src/legacy/core_plugins/kibana/server/tutorials).

## Filebeat module

* [ ] Test log files exist for the grok patterns
* [ ] Generated output for at least 1 log file exists


## Metricbeat module

* [ ] Example `data.json` exists and an automated way to generate it exists (`go test -data`)
* [ ] Test environment in Docker exist for integration tests
