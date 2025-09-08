---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/metricbeat-developer-guide.html
---

# Extending Metricbeat [metricbeat-developer-guide]

Metricbeat periodically interrogates other services to fetch key metrics information. As a developer, you can use Metricbeat in two different ways:

* Extend Metricbeat directly
* Create your own Beat and use Metricbeat as a library

We recommend that you start by creating your own Beat to keep the development of your own module or metricset independent of Metricbeat. At a later stage, if you decide to add a module to Metricbeat, you can reuse the code without making additional changes.

This following topics describe how to contribute to Metricbeat by adding metricsets, modules, and new Beats based on Metricbeat:

* [Overview](./metricbeat-dev-overview.md)
* [Creating a Metricset](./creating-metricsets.md)
* [Metricset Details](./metricset-details.md)
* [Creating a Metricbeat Module](./creating-metricbeat-module.md)
* [Metricbeat Developer FAQ](./dev-faq.md)

If you would like to contribute to Metricbeat or the Beats project, also see [*Contributing to Beats*](./index.md).






