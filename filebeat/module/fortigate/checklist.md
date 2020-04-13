---
name: Filebeat Fortigate Module 
about: "Meta issue to track the creation, updating of a new module or dataset."

---

# Filebeat Fortigate Module

This checklist is intended for Devs which create or update a module to make sure modules are consistent.

## Modules

For a metricset to go GA, the following criterias should be met:

* [*] Supported versions are documented 
* [ ] Documentation
* [*] Fields follow [ECS](https://github.com/elastic/ecs) and [naming conventions](https://www.elastic.co/guide/en/beats/devguide/master/event-conventions.html)
* [ ] Dashboards exists (if applicable)
* [ ] Test log files exist for the grok patterns
* [ ] Generated output for at least 1 log file exists

