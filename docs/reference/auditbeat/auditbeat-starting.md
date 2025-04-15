---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-starting.html
---

# Start Auditbeat [auditbeat-starting]

Before starting Auditbeat:

* Follow the steps in [Quick start: installation and configuration](/reference/auditbeat/auditbeat-installation-configuration.md) to install, configure, and set up the Auditbeat environment.
* Make sure {{kib}} and {{es}} are running.
* Make sure the user specified in `auditbeat.yml` is [authorized to publish events](/reference/auditbeat/privileges-to-publish-events.md).

To start Auditbeat, run:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
sudo service auditbeat start
```


Also see [Auditbeat and systemd](/reference/auditbeat/running-with-systemd.md).
::::::

::::::{tab-item} RPM
```sh
sudo service auditbeat start
```


Also see [Auditbeat and systemd](/reference/auditbeat/running-with-systemd.md).
::::::

::::::{tab-item} MacOS
```sh
sudo chown root auditbeat.yml <1>
sudo ./auditbeat -e
```

1. You’ll be running Auditbeat as root, so you need to change ownership of the configuration file, or run Auditbeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Linux
```sh
sudo chown root auditbeat.yml <1>
sudo ./auditbeat -e
```

1. You’ll be running Auditbeat as root, so you need to change ownership of the configuration file, or run Auditbeat with `--strict.perms=false` specified. See [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::::

::::::{tab-item} Windows
```sh
PS C:\Program Files\auditbeat> Start-Service auditbeat
```

By default, Windows log files are stored in `C:\ProgramData\auditbeat\Logs`.
::::::

:::::::
