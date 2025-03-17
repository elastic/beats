---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/running-with-systemd.html
---

# Auditbeat and systemd [running-with-systemd]

The DEB and RPM packages include a service unit for Linux systems with systemd. On these systems, you can manage Auditbeat by using the usual systemd commands.

The service unit is configured with `UMask=0027` which means the most permissive mask allowed for files created by Auditbeat is `0640`. All configured file permissions higher than `0640` will be ignored. Please edit the unit file manually in case you need to change that.

## Start and stop Auditbeat [_start_and_stop_auditbeat]

Use `systemctl` to start or stop Auditbeat:

```sh
sudo systemctl start auditbeat
```

```sh
sudo systemctl stop auditbeat
```

By default, the Auditbeat service starts automatically when the system boots. To enable or disable auto start use:

```sh
sudo systemctl enable auditbeat
```

```sh
sudo systemctl disable auditbeat
```


## Auditbeat status and logs [_auditbeat_status_and_logs]

To get the service status, use `systemctl`:

```sh
systemctl status auditbeat
```

Logs are stored by default in journald. To view the Logs, use `journalctl`:

```sh
journalctl -u auditbeat.service
```


## Customize systemd unit for Auditbeat [_customize_systemd_unit_for_auditbeat]

The systemd service unit file includes environment variables that you can override to change the default options.

| Variable | Description | Default value |
| --- | --- | --- |
| BEAT_LOG_OPTS | Log options |  |
| BEAT_CONFIG_OPTS | Flags for configuration file path | ``-c /etc/auditbeat/auditbeat.yml`` |
| BEAT_PATH_OPTS | Other paths | ``--path.home /usr/share/auditbeat --path.config /etc/auditbeat --path.data /var/lib/auditbeat --path.logs /var/log/auditbeat`` |

::::{note}
You can use `BEAT_LOG_OPTS` to set debug selectors for logging. However, to configure logging behavior, set the logging options described in [Configure logging](/reference/auditbeat/configuration-logging.md).
::::


To override these variables, create a drop-in unit file in the `/etc/systemd/system/auditbeat.service.d` directory.

For example a file with the following content placed in `/etc/systemd/system/auditbeat.service.d/debug.conf` would override `BEAT_LOG_OPTS` to enable debug for Elasticsearch output.

```text
[Service]
Environment="BEAT_LOG_OPTS=-d elasticsearch"
```

To apply your changes, reload the systemd configuration and restart the service:

```sh
systemctl daemon-reload
systemctl restart auditbeat
```

::::{note}
It is recommended that you use a configuration management tool to include drop-in unit files. If you need to add a drop-in manually, use `systemctl edit auditbeat.service`.
::::



