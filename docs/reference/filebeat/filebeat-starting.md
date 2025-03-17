---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-starting.html
---

# Start Filebeat [filebeat-starting]

Before starting Filebeat:

* Follow the steps in [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md) to install, configure, and set up the Filebeat environment.
* Make sure {{kib}} and {{es}} are running.
* Make sure the user specified in `filebeat.yml` is [authorized to publish events](/reference/filebeat/privileges-to-publish-events.md).

To start Filebeat, run:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
sudo service filebeat start
```

::::{note}
If you use an `init.d` script to start Filebeat, you can’t specify command line flags (see [Command reference](/reference/filebeat/command-line-options.md)). To specify flags, start Filebeat in the foreground.
::::


Also see [Filebeat and systemd](/reference/filebeat/running-with-systemd.md).
::::::

::::::{tab-item} RPM
```sh
sudo service filebeat start
```

::::{note}
If you use an `init.d` script to start Filebeat, you can’t specify command line flags (see [Command reference](/reference/filebeat/command-line-options.md)). To specify flags, start Filebeat in the foreground.
::::


Also see [Filebeat and systemd](/reference/filebeat/running-with-systemd.md).
::::::

::::::{tab-item} MacOS
```sh
sudo chown root filebeat.yml <1>
sudo chown root modules.d/{modulename}.yml <1>
sudo ./filebeat -e
```

1. You’ll be running Filebeat as root, so you need to change ownership of the configuration file and any configurations enabled in the `modules.d` directory, or run Filebeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Linux
```sh
sudo chown root filebeat.yml <1>
sudo chown root modules.d/{modulename}.yml <1>
sudo ./filebeat -e
```

1. You’ll be running Filebeat as root, so you need to change ownership of the configuration file and any configurations enabled in the `modules.d` directory, or run Filebeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Windows
```sh
PS C:\Program Files\filebeat> Start-Service filebeat
```

By default, Windows log files are stored in `C:\ProgramData\filebeat\Logs`.
::::::

:::::::
