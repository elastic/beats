---
navigation_title: "Creating New Kibana Dashboards"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/new-dashboards.html
---

# Creating New Kibana Dashboards for a Beat or a Beat module [new-dashboards]


When contributing to Beats development, you may want to add new dashboards or customize the existing ones. To get started, you can [import the Kibana dashboards](/extend/import-dashboards.md) that come with the official Beats and use them as a starting point for your own dashboards. When you’re done making changes to the dashboards in Kibana, you can use the `export_dashboards` script to [export the dashboards](/extend/export-dashboards.md), along with all dependencies, to a local directory.

To make sure the dashboards are compatible with the latest version of Kibana and Elasticsearch, we recommend that you use the virtual environment under [beats/testing/environments](https://github.com/elastic/beats/tree/master/testing/environments) to import, create, and export the Kibana dashboards.

The following topics provide more detail about importing and working with Beats dashboards:

* [Importing Existing Beat Dashboards](/extend/import-dashboards.md)
* [Building Your Own Beat Dashboards](/extend/build-dashboards.md)
* [Generating the Beat Index Pattern](/extend/generate-index-pattern.md)
* [Exporting New and Modified Beat Dashboards](/extend/export-dashboards.md)
* [Archiving Your Beat Dashboards](/extend/archive-dashboards.md)
* [Sharing Your Beat Dashboards](/extend/share-beat-dashboards.md)







