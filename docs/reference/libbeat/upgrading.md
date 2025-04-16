---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/upgrading.html
---

# Upgrade [upgrading]

This section gives general recommendations for upgrading {{beats}} shippers:

* [Upgrade between minor versions](#upgrade-minor-versions)
* [Upgrade from 8.x to 9.x](#upgrade-8-to-9)
* [Troubleshoot {{beats}} upgrade issues](#troubleshooting-upgrade)

## Upgrade between minor versions [upgrade-minor-versions]

As a general rule, you can upgrade between minor versions (for example, 9.x to 9.y, where x < y) by simply installing the new release and restarting the Beat process. {{beats}} typically maintain backwards compatibility for configuration settings and exported fields. Please review the review the [release notes](docs-content://release-notes/index.md) for potential exceptions.

Upgrading between non-consecutive major versions (e.g. 7.x to 9.x) is not supported.


## Upgrade from 8.x to 9.x [upgrade-8-to-9]

Before upgrading your {{beats}}, review the [release notes](docs-content://release-notes/index.md) and be aware of any documented breaking changes.

If you’re upgrading other products in the stack, also read the {{stack}} [upgrade steps](docs-content://deploy-manage/upgrade/deployment-or-cluster.md).

We recommend that you fully upgrade {{es}} and {{kib}} to version 9.0 before upgrading {{beats}}. The {{beats}} version must be lower than or equal to the {{es}} version. {{beats}} cannot send data to older versions of {{es}}.

If you use the Uptime app in {{kib}}, make sure you add `heartbeat-9*` and `synthetics-*` to **Uptime indices** on the [Settings page](docs-content://solutions/observability/uptime/configure-settings.md). The first index is used by newer versions of a Beat, while the latter is used by {{fleet}}.

::::{important}
Please read through all upgrade steps before proceeding. These steps are required before running the software for the first time.
::::


### Upgrade {{beats}} binaries to 9.0 [upgrade-beats-binaries]

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


### Load 9.0 dashboards [load-9.0-dashboards]

We recommend that you load the 9.x {{kib}} dashboards after upgrading {{kib}} and {{beats}}. That way, you can take advantage of improvements added in 9.0. To load the dashboards, run:

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
```sh subs=true
docker run --rm --net="host" docker.elastic.co/beats/beatname:9.0.0 setup --dashboards
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


### Start your upgraded {{beats}} [start-beats]

After you’ve completed the migration, start the upgraded Beat. Use the command that works with your system.

Check the console and logs for errors.

In {{kib}}, go to **Discover** and verify that events are streaming into {{es}}.


## Troubleshoot {{beats}} upgrade issues [troubleshooting-upgrade]

This section describes common problems you might encounter when upgrading to {{beats}} 9.x.


### {{beats}} is unable to send update or deletion requests to a data stream [unable-to-update-or-delete]

Data streams are designed for use cases where existing data is rarely, if ever, updated. You cannot send update or deletion requests for existing documents directly to a data stream.

If needed, you can update or delete documents by submitting requests directly to the document’s backing index. To learn how, refer to the docs about [using a data stream](docs-content://manage-data/data-store/data-streams/use-data-stream.md).


### Timestamp field is missing [missing-timestamp-field]

{{beats}} requires a timestamp field to send data to data streams. If the timestamp field added by {{beats}} is inadvertently removed by a processor, {{beats}} will be unable to index the event. To fix the problem, modify your processor configuration to avoid removing the timestamp field.


