---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/export-dashboards.html
applies_to:
  stack: ga
---

# Exporting New and Modified Beat Dashboards [export-dashboards]

To export all the dashboards for any Elastic Beat or any community Beat, including any new or modified dashboards and all dependencies such as visualizations, searches, you can use the Go script `export_dashboards.go` from [dev-tools](https://github.com/elastic/beats/tree/main/dev-tools/cmd/dashboards).

Alternatively, if the scripts above are not available, you can use your Beat binary to export dashboards.

## Exporting from Kibana 7.14 or newer [_exporting_from_kibana_7_14_or_newer]

Both the `export_dashboards.go` script and the Beat binary require Kibana 7.14 or newer. You might need to export a single dashboard or all the dashboards available for a module or Beat.


### Export a single Kibana dashboard [_export_a_single_kibana_dashboard]

To export a single dashboard for a module you can use the following command inside a Beat with modules:

```shell
MODULE=redis ID=AV4REOpp5NkDleZmzKkE mage exportDashboard
```

```shell
./filebeat export dashboard --id 7fea2930-478e-11e7-b1f0-cb29bac6bf8b --folder module/redis
```

This generates a `_meta/kibana/<version>` folder under module/redis for the dashboard, where `<version>` is the major version of Kibana, separating assets into dashboards, searches, visualizations, etc. Each exported file is a JSON and their names are the IDs of the assets.

The `setup` command of the Beat imports only the assets in `_meta/kibana/7`, so when you export from Kibana 8.0 or newer, move the exported assets into that folder.

::::{note}
The dashboard ID is available in the dashboard URL. For example, in case the dashboard URL is `app/kibana#/dashboard/AV4REOpp5NkDleZmzKkE?_g=()&_a=(description:'Overview%2...`, the dashboard ID is `AV4REOpp5NkDleZmzKkE`.
::::



### Export all module/Beat dashboards [_export_all_modulebeat_dashboards]

Each module should contain a `module.yml` file with a list of all the dashboards available for the module. For the Beats that don’t have support for modules (e.g. Packetbeat), there is a `dashboards.yml` file that defines all the Packetbeat dashboards.

Below, it’s an example of the `module.yml` file for the system module in Metricbeat:

```shell
dashboards:
- id: Metricbeat-system-overview
  file: Metricbeat-system-overview.ndjson

- id: 79ffd6e0-faa0-11e6-947f-177f697178b8
  file: Metricbeat-host-overview.ndjson

- id: CPU-slash-Memory-per-container
  file: Metricbeat-containers-overview.ndjson
```

Each dashboard is defined by an `id` and the name of ndjson `file` where the dashboard is saved locally.

By passing the yml file to the `export_dashboards.go` script or to the Beat, you can export all the dashboards defined. The `--folder` flag is required, but with `--yml` each dashboard is saved to `_meta/kibana/<version>/dashboard/<file>` in the directory that contains the yml file:

```shell
go run dev-tools/cmd/dashboards/export_dashboards.go --yml filebeat/module/system/module.yml --folder dashboards
```

```shell
./filebeat export dashboard --yml module/system/module.yml --folder module/system
```

Each file is NDJSON that holds the dashboard and all the objects it references. The `setup` command doesn't load these files, so to add dashboards that ship with a module, export each one as described in [Export a single Kibana dashboard](#_export_a_single_kibana_dashboard).


### Export dashboards from a Kibana Space [_export_dashboards_from_a_kibana_space]

If you are using the Kibana Spaces feature and want to export dashboards from a specific Space, pass the Space ID to the `export_dashboards.go` script:

```shell
go run dev-tools/cmd/dashboards/export_dashboards.go -space-id my-space [other-options]
```

In case of running `export dashboard` of a Beat, you need to set the Space ID in `setup.kibana.space.id`.
