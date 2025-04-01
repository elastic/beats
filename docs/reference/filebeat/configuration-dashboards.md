---
navigation_title: "Kibana dashboards"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-dashboards.html
---

# Configure Kibana dashboard loading [configuration-dashboards]


Filebeat comes packaged with example Kibana dashboards, visualizations, and searches for visualizing Filebeat data in Kibana.

To load the dashboards, you can either enable dashboard loading in the `setup.dashboards` section of the `filebeat.yml` config file, or you can run the `setup` command. Dashboard loading is disabled by default.

When dashboard loading is enabled, Filebeat uses the Kibana API to load the sample dashboards. Dashboard loading is only attempted when Filebeat starts up. If Kibana is not available at startup, Filebeat will stop with an error.

To enable dashboard loading, add the following setting to the config file:

```yaml
setup.dashboards.enabled: true
```


## Configuration options [_configuration_options_35]

You can specify the following options in the `setup.dashboards` section of the `filebeat.yml` config file:


### `setup.dashboards.enabled` [_setup_dashboards_enabled]

If this option is set to true, Filebeat loads the sample Kibana dashboards from the local `kibana` directory in the home path of the Filebeat installation.

::::{note}
Filebeat loads dashboards on startup if either `enabled` is set to `true` or the `setup.dashboards` section is included in the configuration.
::::


::::{note}
When dashboard loading is enabled, Filebeat overwrites any existing dashboards that match the names of the dashboards you are loading. This happens every time Filebeat starts.
::::


If no other options are set, the dashboard are loaded from the local `kibana` directory in the home path of the Filebeat installation. To load dashboards from a different location, you can configure one of the following options: [`setup.dashboards.directory`](#directory-option), [`setup.dashboards.url`](#url-option), or [`setup.dashboards.file`](#file-option).


### `setup.dashboards.directory` [directory-option]

The directory that contains the dashboards to load. The default is the `kibana` folder in the home path.


### `setup.dashboards.url` [url-option]

The URL to use for downloading the dashboard archive. If this option is set, Filebeat downloads the dashboard archive from the specified URL instead of using the local directory.


### `setup.dashboards.file` [file-option]

The file archive (zip file) that contains the dashboards to load. If this option is set, Filebeat looks for a dashboard archive in the specified path instead of using the local directory.


### `setup.dashboards.beat` [_setup_dashboards_beat]

In case the archive contains the dashboards for multiple Beats, this setting lets you select the Beat for which you want to load dashboards. To load all the dashboards in the archive, set this option to an empty string. The default is `"filebeat"`.


### `setup.dashboards.kibana_index` [_setup_dashboards_kibana_index]

The name of the Kibana index to use for setting the configuration. The default is `".kibana"`


### `setup.dashboards.index` [_setup_dashboards_index]

The Elasticsearch index name. This setting overwrites the index name defined in the dashboards and index pattern. Example: `"testbeat-*"`

::::{note}
This setting only works for Kibana 6.0 and newer.
::::



### `setup.dashboards.always_kibana` [_setup_dashboards_always_kibana]

Force loading of dashboards using the Kibana API without querying Elasticsearch for the version. The default is `false`.


### `setup.dashboards.retry.enabled` [_setup_dashboards_retry_enabled]

If this option is set to true, and Kibana is not reachable at the time when dashboards are loaded, Filebeat will retry to reconnect to Kibana instead of exiting with an error. Disabled by default.


### `setup.dashboards.retry.interval` [_setup_dashboards_retry_interval]

Duration interval between Kibana connection retries. Defaults to 1 second.


### `setup.dashboards.retry.maximum` [_setup_dashboards_retry_maximum]

Maximum number of retries before exiting with an error. Set to 0 for unlimited retrying. Default is unlimited.


### `setup.dashboards.string_replacements` [_setup_dashboards_string_replacements]

The needle and replacements string map, which is used to replace needle string in dashboards and their references contents.

