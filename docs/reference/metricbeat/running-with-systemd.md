---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/running-with-systemd.html
---

# Metricbeat and systemd [running-with-systemd]

The DEB and RPM packages include a service unit for Linux systems with systemd. On these systems, you can manage Metricbeat by using the usual systemd commands.

The service unit is configured with `UMask=0027` which means the most permissive mask allowed for files created by Metricbeat is `0640`. All configured file permissions higher than `0640` will be ignored. Please edit the unit file manually in case you need to change that.

## Start and stop Metricbeat [_start_and_stop_metricbeat]

Use `systemctl` to start or stop Metricbeat:

```sh
sudo systemctl start metricbeat
```

```sh
sudo systemctl stop metricbeat
```

By default, the Metricbeat service starts automatically when the system boots. To enable or disable auto start use:

```sh
sudo systemctl enable metricbeat
```

```sh
sudo systemctl disable metricbeat
```


## Metricbeat status and logs [_metricbeat_status_and_logs]

To get the service status, use `systemctl`:

```sh
systemctl status metricbeat
```

Logs are stored by default in journald. To view the Logs, use `journalctl`:

```sh
journalctl -u metricbeat.service
```


## Customize systemd unit for Metricbeat [_customize_systemd_unit_for_metricbeat]

The systemd service unit file includes environment variables that you can override to change the default options.

| Variable | Description | Default value |
| --- | --- | --- |
| BEAT_LOG_OPTS | Log options |  |
| BEAT_CONFIG_OPTS | Flags for configuration file path | ``-c /etc/metricbeat/metricbeat.yml`` |
| BEAT_PATH_OPTS | Other paths | ``--path.home /usr/share/metricbeat --path.config /etc/metricbeat --path.data /var/lib/metricbeat --path.logs /var/log/metricbeat`` |

::::{note}
You can use `BEAT_LOG_OPTS` to set debug selectors for logging. However, to configure logging behavior, set the logging options described in [Configure logging](/reference/metricbeat/configuration-logging.md).
::::


To override these variables, create a drop-in unit file in the `/etc/systemd/system/metricbeat.service.d` directory.

For example a file with the following content placed in `/etc/systemd/system/metricbeat.service.d/debug.conf` would override `BEAT_LOG_OPTS` to enable debug for Elasticsearch output.

```text
[Service]
Environment="BEAT_LOG_OPTS=-d elasticsearch"
```

To apply your changes, reload the systemd configuration and restart the service:

```sh
systemctl daemon-reload
systemctl restart metricbeat
```

::::{note}
It is recommended that you use a configuration management tool to include drop-in unit files. If you need to add a drop-in manually, use `systemctl edit metricbeat.service`.
::::



