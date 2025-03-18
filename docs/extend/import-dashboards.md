---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/import-dashboards.html
---

# Importing Existing Beat Dashboards [import-dashboards]

The official Beats come with Kibana dashboards, and starting with 6.0.0, they are part of every Beat package.

You can use the Beat executable to import all the dashboards and the index pattern for a Beat, including the dependencies such as visualizations and searches.

To import the dashboards, run the `setup` command.

```shell
./metricbeat setup
```

The `setup` phase loads several dependencies, such as:

* Index mapping template in Elasticsearch
* Kibana dashboards
* Ingest pipelines
* ILM policy

The dependencies vary depending on the Beat you’re setting up.

For more details about the `setup` command, see the command-line help. For example:

```shell
./metricbeat help setup

This command does initial setup of the environment:

 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * Kibana dashboards (where available).
 * ML jobs (where available).
 * Ingest pipelines (where available).
 * ILM policy (for Elasticsearch 6.5 and newer).

Usage:
  metricbeat setup [flags]

Flags:
      --dashboards         Setup dashboards
  -h, --help               help for setup
      --index-management   Setup all components related to Elasticsearch index management, including template, ilm policy and rollover alias
      --pipelines          Setup Ingest pipelines
```

The flags are useful when you don’t want to load everything. For example, to import only the dashboards, use the `--dashboards` flag:

```shell
./metricbeat setup --dashboards
```

Starting with Beats 6.0.0, the dashboards are no longer loaded directly into Elasticsearch. Instead, they are imported directly into Kibana. Thus, if your Kibana instance is not listening on localhost, or you enabled {{xpack}} for Kibana, you need to either configure the Kibana endpoint in the config for the Beat, or pass the Kibana host and credentials as arguments to the `setup` command. For example:

```shell
./metricbeat setup -E setup.kibana.host=192.168.3.206:5601 -E setup.kibana.username=elastic -E setup.kibana.password=secret
```

By default, the `setup` command imports the dashboards from the `kibana` directory, which is available in the Beat package.

::::{note}
The format of the saved dashboards is not compatible between Kibana 5.x and 6.x. Thus, the Kibana 5.x dashboards are available in the `5.x` directory, and the Kibana 6.0 dashboards, and older are in the `default` directory.
::::


In case you are using customized dashboards, you can import them:

* from a local directory:

    ```shell
    ./metricbeat setup -E setup.dashboards.directory=kibana
    ```

* from a local zip archive:

    ```shell
    ./metricbeat setup -E setup.dashboards.file=metricbeat-dashboards-6.0.zip
    ```

* from a zip archive available online:

    ```shell
    ./metricbeat setup -E setup.dashboards.url=path/to/url
    ```

    See [Kibana dashboards configuration](#import-dashboard-options) for a description of the `setup.dashboards` configuration options.


## Import Dashboards for Development [import-dashboards-for-development]

You can make use of the Magefile from the Beat GitHub repository to import the dashboards. If Kibana is running on localhost, then you can run the following command from the root of the Beat:

```shell
mage dashboards
```


## Kibana dashboards configuration [import-dashboard-options]

The configuration file (`*.reference.yml`) of each Beat contains the `setup.dashboards` section for configuring from where to get the Kibana dashboards, as well as the name of the index pattern. Each of these configuration options can be overwritten with the command line options by using `-E` flag.

**`setup.dashboards.directory=<local_dir>`**
:   Local directory that contains the saved dashboards and their dependencies. The default value is the `kibana` directory available in the Beat package.

**`setup.dashboards.file=<local_archive>`**
:   Local zip archive with the dashboards. The archive can contain Kibana dashboards for a single Beat or for multiple Beats. The dashboards of each Beat are placed under a separate directory with the name of the Beat.

**`setup.dashboards.url=<zip_url>`**
:   Zip archive with the dashboards, available online. The archive can contain Kibana dashboards for a single Beat or for multiple Beats. The dashboards for each Beat are placed under a separate directory with the name of the Beat.

**`setup.dashboards.index <elasticsearch_index>`**
:   You should only use this option if you want to change the index pattern name that’s used by default. For example, if the default is `metricbeat-*`, you can change it to `custombeat-*`.


