---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/running-with-systemd.html
---

# Heartbeat and systemd [running-with-systemd]

The DEB and RPM packages include a service unit for Linux systems with systemd. On these systems, you can manage Heartbeat by using the usual systemd commands.

The service unit is configured with `UMask=0027` which means the most permissive mask allowed for files created by Heartbeat is `0640`. All configured file permissions higher than `0640` will be ignored. Please edit the unit file manually in case you need to change that.

## Start and stop Heartbeat [_start_and_stop_heartbeat]

Use `systemctl` to start or stop Heartbeat:

```sh
sudo systemctl start heartbeat-elastic
```

```sh
sudo systemctl stop heartbeat-elastic
```

By default, the Heartbeat service starts automatically when the system boots. To enable or disable auto start use:

```sh
sudo systemctl enable heartbeat-elastic
```

```sh
sudo systemctl disable heartbeat-elastic
```


## Heartbeat status and logs [_heartbeat_status_and_logs]

To get the service status, use `systemctl`:

```sh
systemctl status heartbeat-elastic
```

Logs are stored by default in journald. To view the Logs, use `journalctl`:

```sh
journalctl -u heartbeat-elastic.service
```


## Customize systemd unit for Heartbeat [_customize_systemd_unit_for_heartbeat]

The systemd service unit file includes environment variables that you can override to change the default options.

| Variable | Description | Default value |
| --- | --- | --- |
| BEAT_LOG_OPTS | Log options |  |
| BEAT_CONFIG_OPTS | Flags for configuration file path | ``-c /etc/heartbeat/heartbeat.yml`` |
| BEAT_PATH_OPTS | Other paths | ``--path.home /usr/share/heartbeat --path.config /etc/heartbeat --path.data /var/lib/heartbeat --path.logs /var/log/heartbeat`` |

::::{note}
You can use `BEAT_LOG_OPTS` to set debug selectors for logging. However, to configure logging behavior, set the logging options described in [Configure logging](/reference/heartbeat/configuration-logging.md).
::::


To override these variables, create a drop-in unit file in the `/etc/systemd/system/heartbeat-elastic.service.d` directory.

For example a file with the following content placed in `/etc/systemd/system/heartbeat-elastic.service.d/debug.conf` would override `BEAT_LOG_OPTS` to enable debug for Elasticsearch output.

```text
[Service]
Environment="BEAT_LOG_OPTS=-d elasticsearch"
```

To apply your changes, reload the systemd configuration and restart the service:

```sh
systemctl daemon-reload
systemctl restart heartbeat-elastic
```

::::{note}
It is recommended that you use a configuration management tool to include drop-in unit files. If you need to add a drop-in manually, use `systemctl edit heartbeat-elastic.service`.
::::



