---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/directory-layout.html
---

# Directory layout [directory-layout]

The directory layout of an installation is as follows:

::::{tip}
Archive installation has a different layout. See [zip, tar.gz, or tgz](#directory-layout-archive).
::::


| Type | Description | Default Location | Config Option |
| --- | --- | --- | --- |
| home | Home of the Winlogbeat installation. |  | `path.home` |
| bin | The location for the binary files. | `{path.home}/bin` |  |
| config | The location for configuration files. | `{path.home}` | `path.config` |
| data | The location for persistent data files. | `{path.home}/data` | `path.data` |
| logs | The location for the logs created by Winlogbeat. | `{path.home}/logs` | `path.logs` |

You can change these settings by using CLI flags or setting [path options](/reference/winlogbeat/configuration-path.md) in the configuration file.

## Default paths [_default_paths]

Winlogbeat uses the following default paths unless you explicitly change them.


#### zip, tar.gz, or tgz [directory-layout-archive]

| Type | Description | Location |
| --- | --- | --- |
| home | Home of the Winlogbeat installation. | `{extract.path}` |
| bin | The location for the binary files. | `{extract.path}` |
| config | The location for configuration files. | `{extract.path}` |
| data | The location for persistent data files. | `{extract.path}/data` |
| logs | The location for the logs created by Winlogbeat. | `{extract.path}/logs` |

For the zip, tar.gz, or tgz distributions, these paths are based on the location of the extracted binary file. This means that if you start Winlogbeat with the following simple command, all paths are set correctly:

```sh
Start-Service winlogbeat
```


