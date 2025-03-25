---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/build-dashboards.html
---

# Building Your Own Beat Dashboards [build-dashboards]

::::{note}
If you want to modify a dashboard that comes with a Beat, it’s better to modify a copy of the dashboard because the Beat overwrites the dashboards during the setup phase in order to have the latest version. For duplicating a dashboard, just use the `Clone` button from the top of the page.
::::


Before building your own dashboards or customizing the existing ones, you need to load:

* the Beat index pattern, which specifies how Kibana should display the Beat fields
* the Beat dashboards that you want to customize

For the Elastic Beats, the index pattern is available in the Beat package under `kibana/*/index-pattern`. The index-pattern is automatically generated from the `fields.yml` file, available in the Beat package. For more details check the [generate index pattern](/extend/generate-index-pattern.md) section.

All Beats dashboards, visualizations and saved searches must follow common naming conventions:

* Dashboard names have prefix `[BeatName Module]`, e.g. `[Filebeat Nginx] Access logs`
* Visualizations and searches have suffix `[BeatName Module]`, e.g. `Top processes [Filebeat Nginx]`

::::{note}
You can set a custom name (skip suffix) for visualization placed on a dashboard. The original visualization will stay intact.
::::


The naming convention rules can be verified with the the tool `mage check`. The command fails if it detects:

* empty description on a dashboard
* unexpected dashboard title format (missing prefix `[BeatName ModuleName]`)
* unexpected visualization title format (missing suffix `[BeatName Module]`)

After creating your own dashboards in Kibana, you can [export the Kibana dashboards](/extend/export-dashboards.md) to a local directory, and then [archive the dashboards](/extend/archive-dashboards.md) in order to be able to share the dashboards with the community.

