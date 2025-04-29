---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/running-with-systemd.html
---

# Packetbeat and systemd [running-with-systemd]

The DEB and RPM packages include a service unit for Linux systems with systemd. On these systems, you can manage Packetbeat by using the usual systemd commands.

The service unit is configured with `UMask=0027` which means the most permissive mask allowed for files created by Packetbeat is `0640`. All configured file permissions higher than `0640` will be ignored. Please edit the unit file manually in case you need to change that.

## Start and stop Packetbeat [_start_and_stop_packetbeat]

Use `systemctl` to start or stop Packetbeat:

```sh
sudo systemctl start packetbeat
```

```sh
sudo systemctl stop packetbeat
```

By default, the Packetbeat service starts automatically when the system boots. To enable or disable auto start use:

```sh
sudo systemctl enable packetbeat
```

```sh
sudo systemctl disable packetbeat
```


## Packetbeat status and logs [_packetbeat_status_and_logs]

To get the service status, use `systemctl`:

```sh
systemctl status packetbeat
```

Logs are stored by default in journald. To view the Logs, use `journalctl`:

```sh
journalctl -u packetbeat.service
```


## Customize systemd unit for Packetbeat [_customize_systemd_unit_for_packetbeat]

The systemd service unit file includes environment variables that you can override to change the default options.

| Variable | Description | Default value |
| --- | --- | --- |
| BEAT_LOG_OPTS | Log options |  |
| BEAT_CONFIG_OPTS | Flags for configuration file path | ``-c /etc/packetbeat/packetbeat.yml`` |
| BEAT_PATH_OPTS | Other paths | ``--path.home /usr/share/packetbeat --path.config /etc/packetbeat --path.data /var/lib/packetbeat --path.logs /var/log/packetbeat`` |

::::{note}
You can use `BEAT_LOG_OPTS` to set debug selectors for logging. However, to configure logging behavior, set the logging options described in [Configure logging](/reference/packetbeat/configuration-logging.md).
::::


To override these variables, create a drop-in unit file in the `/etc/systemd/system/packetbeat.service.d` directory.

For example a file with the following content placed in `/etc/systemd/system/packetbeat.service.d/debug.conf` would override `BEAT_LOG_OPTS` to enable debug for Elasticsearch output.

```text
[Service]
Environment="BEAT_LOG_OPTS=-d elasticsearch"
```

To apply your changes, reload the systemd configuration and restart the service:

```sh
systemctl daemon-reload
systemctl restart packetbeat
```

::::{note}
It is recommended that you use a configuration management tool to include drop-in unit files. If you need to add a drop-in manually, use `systemctl edit packetbeat.service`.
::::



