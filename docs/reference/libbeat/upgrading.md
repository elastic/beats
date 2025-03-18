---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/upgrading.html
---

# Upgrade [upgrading]

This section gives general recommendations for upgrading {{beats}} shippers:

* [Upgrade between minor versions](#upgrade-minor-versions)
* [Upgrade from 7.x to 8.x](#upgrade-7-to-8)
* [Troubleshoot {{beats}} upgrade issues](#troubleshooting-upgrade)

If you’re upgrading other products in the stack, also read the [Elastic Stack Installation and Upgrade Guide](docs-content://deploy-manage/index.md).


## Upgrade between minor versions [upgrade-minor-versions]

As a general rule, you can upgrade between minor versions (for example, 8.x to 8.y, where x < y) by simply installing the new release and restarting the Beat process. {{beats}} typically maintain backwards compatibility for configuration settings and exported fields. Please review the [release notes](/release-notes/index.md) for potential exceptions.

Upgrading between non-consecutive major versions (e.g. 6.x to 8.x) is not supported.


## Upgrade from 7.x to 8.x [upgrade-7-to-8]

Before upgrading your {{beats}}, review the [breaking changes](/release-notes/breaking-changes.md) and the [*Release notes*](/release-notes/index.md).

If you’re upgrading other products in the stack, also read the [Elastic Stack Installation and Upgrade Guide](docs-content://deploy-manage/index.md).

We recommend that you fully upgrade {{es}} and {{kib}} to version 8.0 before upgrading {{beats}}. The {{beats}} version must be lower than or equal to the {{es}} version. {{beats}} cannot send data to older versions of {{es}}.

If you use the Uptime app in {{kib}}, make sure you add `heartbeat-8*` and `synthetics-*` to **Uptime indices** on the [Settings page](docs-content://solutions/observability/apps/configure-settings.md). The first index is used by newer versions of a Beat, while the latter is used by {{fleet}}.

If you’re on {{beats}} 7.0 through 7.16, upgrade the {{stack}} and {{beats}} to version 7.17 **before** proceeding with the 8.0 upgrade.

Upgrading between non-consecutive major versions (e.g. 6.x to 8.x) is not supported.

::::{important}
Please read through all upgrade steps before proceeding. These steps are required before running the software for the first time.
::::



### Upgrade to {{beats}} 7.17 before upgrading to 8.0 [upgrade-to-7.17]

The upgrade procedure assumes that you have {{beats}} 7.17 installed. If you’re on a previous 7.x version of {{beats}}, **upgrade to version 7.17 first**. If you’re using other products in the {{stack}}, upgrade {{beats}} as part of the [{{stack}} upgrade process](docs-content://deploy-manage/upgrade/deployment-or-cluster.md).

After upgrading to 7.17, go to **Index Management** in {{kib}} and verify that the 7.17 index template has been loaded into {{es}}.

:::{image} images/confirm-index-template.png
:alt: Screen capture showing that metricbeat-1.17.0 index template is loaded
:class: screenshot
:::

If the 7.17 index template is not loaded, load it now.

If you created custom dashboards prior to version 7.17, you must upgrade them to 7.17 before proceeding. Otherwise, the dashboards will stop working because {{kib}} no longer provides the API used for dashboards in 7.x.


### Upgrade {{beats}} binaries to 8.0 [upgrade-beats-binaries]

Before upgrading:

1. Stop the existing {{beats}} process by using the appropriate command for your system.
2. Back up the `data` and `config` directories by copying them to another location.

    ::::{tip}
    The location of these directories depends on the installation method. To see the current paths, start the Beat from a terminal, and the `data` and `config` paths are printed at startup.
    ::::


To upgrade using a Debian or RPM package:

* Use `rpm` or `dpkg` to install the new package. All files are installed in the appropriate location for the operating system and {{beats}} config files are not overwritten.

To upgrade using a zip or compressed tarball:

1. Extract the zip or tarball to a *new* directory. This is critical if you are not using external `config` and `data` directories.
2. Set the following options in the {{beats}} configuration file:

    * Set `path.config` to point to your external `config` directory. If you are not using an external `config` directory, copy your old configuration over to the new installation.
    * Set `path.data` to point to your external data directory. If you are not using an external `data` directory, copy your old data directory over to the new installation.
    * Set `path.logs` to point to the location where you want to store your logs. If you do not specify this setting, logs are stored in the directory you extracted the archive to.


Complete the upgrade tasks described in the following sections **before** restarting the {{beats}} process.


### Migrate configuration files [migrate-config-files]

{{beats}} 8.0 comes with several backwards incompatible configuration changes. Before upgrading, review the [8.0](https://www.elastic.co/guide/en/beats/libbeat/8.x/breaking-changes-8.0.html) document. Also review the full list of breaking changes in the [Beats version 8.0.0](https://www.elastic.co/guide/en/beats/libbeat/8.x/release-notes-8.0.0.html).

Where possible, we kept the old configuration options working, but deprecated them. However, deprecation was not always possible, so if you use any of the settings described under breaking changes, make sure you understand the alternatives that we provide.


### Load the 8.0 {{es}} index templates [upgrade-index-template]

Starting in version 8.0, the default {{es}} index templates configure data streams instead of traditional {{es}} indices. Data streams are optimized for storing append-only time series data. They are well-suited for logs, events, metrics, and other continuously generated data. However, unlike traditional {{es}} indices, data streams support create operations only; they do not support update and delete operations.

To use data streams, load the default index templates **before** ingesting any data into {{es}}.

:::::::{tab-set}

::::::{tab-item} DEB
```sh
beatname setup --index-management
```
::::::

::::::{tab-item} RPM
```sh
beatname setup --index-management
```
::::::

::::::{tab-item} MacOS
```sh
./beatname setup --index-management
```
::::::

::::::{tab-item} Linux
```sh
./beatname setup --index-management
```
::::::

::::::{tab-item} Docker
```sh
docker run --rm --net="host" docker.elastic.co/beats/beatname:9.0.0-beta1 setup --index-management
```
::::::

::::::{tab-item} Windows
Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed a Beat, and run:

```sh
PS > .\beatname.exe setup --index-management
```
::::::

:::::::
If you’re not collecting time series data, you can continue to use {{beats}} to send data to aliases and indices. To do this, create a custom index template and load it manually. To learn more about creating index templates, refer to [Index templates](docs-content://manage-data/data-store/templates.md).


### Load 8.0 dashboards [load-8.0-dashboards]

We recommend that you load the 8.0 {{kib}} dashboards after upgrading {{kib}} and {{beats}}. That way, you can take advantage of improvements added in 8.0. To load the dashboards, run:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
beatname setup --dashboards
```
::::::

::::::{tab-item} RPM
```sh
beatname setup --dashboards
```
::::::

::::::{tab-item} MacOS
```sh
./beatname setup --dashboards
```
::::::

::::::{tab-item} Linux
```sh
./beatname setup --dashboards
```
::::::

::::::{tab-item} Docker
```sh
docker run --rm --net="host" docker.elastic.co/beats/beatname:9.0.0-beta1 setup --dashboards
```
::::::

::::::{tab-item} Windows
Open a PowerShell prompt as an Administrator (right-click the PowerShell icon and select **Run As Administrator**).

From the PowerShell prompt, change to the directory where you installed a Beat, and run:

```sh
PS > .\beatname.exe setup --dashboards
```
::::::

:::::::

### Migrate custom dashboards and visualizations [migrate-custom-dashboards]

All Elastic {{beats}} send events with ECS-compliant field names. If you have any custom {{kib}} dashboards or visualizations that use old fields, adjust them now to use ECS field names.

To learn more about ECS, refer to the [ECS overview](ecs://reference/index.md).

::::{note}
If you enabled the compatibility layer in 7.x (that is, if you set `migration.6_to_7.enabled: true`), make sure your custom dashboards no longer rely on the old aliases created by that setting. The old aliases are no longer supported. They may continue to work, but will be removed without notice in a future release.
::::



### Start your upgraded {{beats}} [start-beats]

After you’ve completed the migration, start the upgraded Beat. Use the command that works with your system.

Check the console and logs for errors.

In {{kib}}, go to **Discover** and verify that events are streaming into {{es}}.


## Troubleshoot {{beats}} upgrade issues [troubleshooting-upgrade]

This section describes common problems you might encounter when upgrading to {{beats}} 8.x.

You can avoid some of these problems by reading [Upgrade from 7.x to 8.x](#upgrade-7-to-8) before upgrading {{beats}}.


### {{beats}} is unable to send update or deletion requests to a data stream [unable-to-update-or-delete]

Data streams are designed for use cases where existing data is rarely, if ever, updated. You cannot send update or deletion requests for existing documents directly to a data stream.

If needed, you can update or delete documents by submitting requests directly to the document’s backing index. To learn how, refer to the docs about [using a data stream](docs-content://manage-data/data-store/data-streams/use-data-stream.md).


### Timestamp field is missing [missing-timestamp-field]

{{beats}} requires a timestamp field to send data to data streams. If the timestamp field added by {{beats}} is inadvertently removed by a processor, {{beats}} will be unable to index the event. To fix the problem, modify your processor configuration to avoid removing the timestamp field.


### Missing fields or too many fields in the index [missing-fields]

You may have run the Beat before loading the required index template. To clean up and start again:

1. Delete the index that was created when you ran the Beat. For example:

    ```sh
    DELETE metricbeat-9.0.0-beta1-2019.04.02*
    ```

    ::::{warning}
    Be careful when using wildcards to delete indices. Make sure the pattern matches only the indices you want to delete. The example shown here deletes all data indexed into the metricbeat-9.0.0-beta1 indices on 2019.04.02.
    ::::

2. Delete the index template that was loaded earlier. For example:

    ```sh
    DELETE /_index_template/metricbeat-9.0.0-beta1
    ```

3. Load the correct index template. See [Load the 8.0 {{es}} index templates](#upgrade-index-template).
4. Restart {{beats}}.

