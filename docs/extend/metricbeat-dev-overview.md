---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/metricbeat-dev-overview.html
---

# Overview [metricbeat-dev-overview]

Metricbeat consists of modules and metricsets. A Metricbeat module is typically named after the service the metrics are fetched from, such as redis, mysql, and so on. Each module can contain multiple metricsets. A metricset represents multiple metrics that are normally retrieved with one request from the remote system. For example, the Redis `info` metricset retrieves info that you get when you run the Redis `INFO` command, and the MySQL `status` metricset retrieves info that you get when you issue the MySQL `SHOW GLOBAL STATUS` query.


## Module and Metricsets Requirements [_module_and_metricsets_requirements]

To guarantee the best user experience, it’s important to us that only high quality modules are part of Metricbeat. The modules and metricsets that are contributed must meet the following requirements:

* Complete `fields.yml` file to generate docs and Elasticsearch templates
* Documentation files
* Integration tests
* 80% test coverage (unit, integration, and system tests combined)

Metricbeat allows you to build a wide variety of modules and metricsets on top of it. For a module to be accepted, it should focus on fetching service metrics directly from the service itself and not via a third-party tool. The goal is to have as few movable parts as possible and for Metricbeat to run as close as possible to the service that it needs to monitor.

