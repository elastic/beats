---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/packetbeat-starting.html
---

# Start Packetbeat [packetbeat-starting]

Before starting Packetbeat:

* Follow the steps in [Quick start: installation and configuration](/reference/packetbeat/packetbeat-installation-configuration.md) to install, configure, and set up the Packetbeat environment.
* Make sure {{kib}} and {{es}} are running.
* Make sure the user specified in `packetbeat.yml` is [authorized to publish events](/reference/packetbeat/privileges-to-publish-events.md).

To start Packetbeat, run:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
sudo service packetbeat start
```

::::{note}
If you use an `init.d` script to start Packetbeat, you can’t specify command line flags (see [Command reference](/reference/packetbeat/command-line-options.md)). To specify flags, start Packetbeat in the foreground.
::::


Also see [Packetbeat and systemd](/reference/packetbeat/running-with-systemd.md).
::::::

::::::{tab-item} RPM
```sh
sudo service packetbeat start
```

::::{note}
If you use an `init.d` script to start Packetbeat, you can’t specify command line flags (see [Command reference](/reference/packetbeat/command-line-options.md)). To specify flags, start Packetbeat in the foreground.
::::


Also see [Packetbeat and systemd](/reference/packetbeat/running-with-systemd.md).
::::::

::::::{tab-item} MacOS
```sh
sudo chown root packetbeat.yml <1>
sudo ./packetbeat -e
```

1. You’ll be running Packetbeat as root, so you need to change ownership of the configuration file, or run Packetbeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Linux
```sh
sudo chown root packetbeat.yml <1>
sudo ./packetbeat -e
```

1. You’ll be running Packetbeat as root, so you need to change ownership of the configuration file, or run Packetbeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Windows
```sh
PS C:\Program Files\packetbeat> Start-Service packetbeat
```

By default, Windows log files are stored in `C:\ProgramData\packetbeat\Logs`.
::::::

:::::::
