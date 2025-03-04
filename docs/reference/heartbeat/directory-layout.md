---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/directory-layout.html
---

# Directory layout [directory-layout]

The directory layout of an installation is as follows:

::::{tip}
Archive installation has a different layout. See [zip, tar.gz, or tgz](#directory-layout-archive).
::::


| Type | Description | Default Location | Config Option |
| --- | --- | --- | --- |
| home | Home of the Heartbeat installation. |  | `path.home` |
| bin | The location for the binary files. | `{path.home}/bin` |  |
| config | The location for configuration files. | `{path.home}` | `path.config` |
| data | The location for persistent data files. | `{path.home}/data` | `path.data` |
| logs | The location for the logs created by Heartbeat. | `{path.home}/logs` | `path.logs` |

You can change these settings by using CLI flags or setting [path options](/reference/heartbeat/configuration-path.md) in the configuration file.

## Default paths [_default_paths]

Heartbeat uses the following default paths unless you explicitly change them.


#### deb and rpm [_deb_and_rpm]

| Type | Description | Location |
| --- | --- | --- |
| home | Home of the Heartbeat installation. | `/usr/share/heartbeat` |
| bin | The location for the binary files. | `/usr/share/heartbeat/bin` |
| config | The location for configuration files. | `/etc/heartbeat` |
| data | The location for persistent data files. | `/var/lib/heartbeat` |
| logs | The location for the logs created by Heartbeat. | `/var/log/heartbeat` |

For the deb and rpm distributions, these paths are set in the init script or in the systemd unit file.  Make sure that you start the Heartbeat service by using the preferred operating system method (init scripts or `systemctl`). Otherwise the paths might be set incorrectly.


#### docker [_docker]

| Type | Description | Location |
| --- | --- | --- |
| home | Home of the Heartbeat installation. | `/usr/share/heartbeat` |
| bin | The location for the binary files. | `/usr/share/heartbeat` |
| config | The location for configuration files. | `/usr/share/heartbeat` |
| data | The location for persistent data files. | `/usr/share/heartbeat/data` |
| logs | The location for the logs created by Heartbeat. | `/usr/share/heartbeat/logs` |


#### zip, tar.gz, or tgz [directory-layout-archive]

| Type | Description | Location |
| --- | --- | --- |
| home | Home of the Heartbeat installation. | `{extract.path}` |
| bin | The location for the binary files. | `{extract.path}` |
| config | The location for configuration files. | `{extract.path}` |
| data | The location for persistent data files. | `{extract.path}/data` |
| logs | The location for the logs created by Heartbeat. | `{extract.path}/logs` |

For the zip, tar.gz, or tgz distributions, these paths are based on the location of the extracted binary file. This means that if you start Heartbeat with the following simple command, all paths are set correctly:

```sh
./heartbeat
```


