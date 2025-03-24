---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/export-dashboards.html
---

# Exporting New and Modified Beat Dashboards [export-dashboards]

To export all the dashboards for any Elastic Beat or any community Beat, including any new or modified dashboards and all dependencies such as visualizations, searches, you can use the Go script `export_dashboards.go` from [dev-tools](https://github.com/elastic/beats/tree/master/dev-tools/cmd/dashboards). See the dev-tools [readme](https://github.com/elastic/beats/tree/master/dev-tools/README.md) for more info.

Alternatively, if the scripts above are not available, you can use your Beat binary to export Kibana 6.0 dashboards or later.

## Exporting from Kibana 6.0 to 7.14 [_exporting_from_kibana_6_0_to_7_14]

The `dev-tools/cmd/export_dashboards.go` script helps you export your customized Kibana dashboards until the v7.14.x release. You might need to export a single dashboard or all the dashboards available for a module or Beat.

It is also possible to use a Beat binary to export.


## Exporting from Kibana 7.15 or newer [_exporting_from_kibana_7_15_or_newer]

From 7.15, your Beats version must be the same as your Kibana version to make sure the export API required is available.

### Migrate legacy dashboards made with Kibana 7.14 or older [_migrate_legacy_dashboards_made_with_kibana_7_14_or_older]

After you updated your Kibana instance to at least 7.15, you have to export your dashboards again with either `export_dashboards.go` tool or with your Beat.


### Export a single Kibana dashboard [_export_a_single_kibana_dashboard]

To export a single dashboard for a module you can use the following command inside a Beat with modules:

```shell
MODULE=redis ID=AV4REOpp5NkDleZmzKkE mage exportDashboard
```

```shell
./filebeat export dashboard --id 7fea2930-478e-11e7-b1f0-cb29bac6bf8b --folder module/redis
```

This generates an appropriate folder under module/redis for the dashboard, separating assets into dashboards, searches, vizualizations, etc. Each exported file is a JSON and their names are the IDs of the assets.

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

By passing the yml file to the `export_dashboards.go` script or to the Beat, you can export all the dashboards defined:

```shell
go run dev-tools/cmd/dashboards/export_dashboards.go --yml filebeat/module/system/module.yml --folder dashboards
```

```shell
./filebeat export dashboard --yml filebeat/module/system/module.yml
```


### Export dashboards from a Kibana Space [_export_dashboards_from_a_kibana_space]

If you are using the Kibana Spaces feature and want to export dashboards from a specific Space, pass the Space ID to the `export_dashboards.go` script:

```shell
go run dev-tools/cmd/dashboards/export_dashboards.go -space-id my-space [other-options]
```

In case of running `export dashboard` of a Beat, you need to set the Space ID in `setup.kibana.space.id`.



## Exporting Kibana 5.x dashboards [_exporting_kibana_5_x_dashboards]

To export only some Kibana dashboards for an Elastic Beat or community Beat, you can simply pass a regular expression to the `export_dashboards.py` script to match the selected Kibana dashboards.

Before running the `export_dashboards.py` script for the first time, you need to create an environment that contains all the required Python packages.

```shell
make python-env
```

For example, to export all Kibana dashboards that start with the **Packetbeat** name:

```shell
python ../dev-tools/cmd/dashboards/export_dashboards.py --regex Packetbeat*
```

To see all the available options, read the descriptions below or run:

```shell
python ../dev-tools/cmd/dashboards/export_dashboards.py -h
```

**`--url <elasticsearch_url>`**
:   The Elasticsearch URL. The default value is [http://localhost:9200](http://localhost:9200).

**`--regex <regular_expression>`**
:   Regular expression to match all the Kibana dashboards to be exported. This argument is required.

**`--kibana <kibana_index>`**
:   The Elasticsearch index pattern where Kibana saves its configuration. The default value is `.kibana`.

**`--dir <output_dir>`**
:   The output directory where the dashboards and all dependencies will be saved. The default value is `output`.

The output directory has the following structure:

```shell
output/
    index-pattern/
    dashboard/
    visualization/
    search/
```
