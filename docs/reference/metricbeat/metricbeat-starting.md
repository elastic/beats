---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-starting.html
---

# Start Metricbeat [metricbeat-starting]

Before starting Metricbeat:

* Follow the steps in [Quick start: installation and configuration](/reference/metricbeat/metricbeat-installation-configuration.md) to install, configure, and set up the Metricbeat environment.
* Make sure {{kib}} and {{es}} are running.
* Make sure the user specified in `metricbeat.yml` is [authorized to publish events](/reference/metricbeat/privileges-to-publish-events.md).

To start Metricbeat, run:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
sudo service metricbeat start
```

::::{note}
If you use an `init.d` script to start Metricbeat, you can’t specify command line flags (see [Command reference](/reference/metricbeat/command-line-options.md)). To specify flags, start Metricbeat in the foreground.
::::


Also see [Metricbeat and systemd](/reference/metricbeat/running-with-systemd.md).
::::::

::::::{tab-item} RPM
```sh
sudo service metricbeat start
```

::::{note}
If you use an `init.d` script to start Metricbeat, you can’t specify command line flags (see [Command reference](/reference/metricbeat/command-line-options.md)). To specify flags, start Metricbeat in the foreground.
::::


Also see [Metricbeat and systemd](/reference/metricbeat/running-with-systemd.md).
::::::

::::::{tab-item} MacOS
```sh
sudo chown root metricbeat.yml <1>
sudo chown root modules.d/{modulename}.yml <1>
sudo ./metricbeat -e
```

1. You’ll be running Metricbeat as root, so you need to change ownership of the configuration file and any configurations enabled in the `modules.d` directory, or run Metricbeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Linux
```sh
sudo chown root metricbeat.yml <1>
sudo chown root modules.d/{modulename}.yml <1>
sudo ./metricbeat -e
```

1. You’ll be running Metricbeat as root, so you need to change ownership of the configuration file and any configurations enabled in the `modules.d` directory, or run Metricbeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Windows
```sh
PS C:\Program Files\metricbeat> Start-Service metricbeat
```

By default, Windows log files are stored in `C:\ProgramData\metricbeat\Logs`.

::::{note}
On Windows, statistics about system load and swap usage are currently not captured
::::
::::::

:::::::
