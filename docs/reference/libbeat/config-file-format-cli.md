---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/config-file-format-cli.html
---

# Command line arguments [config-file-format-cli]

Config files to load are set using the `-c` flag on command line. If no flag is given, a beat and OS-specific default file path will be assumed.

You can specify multiple configuration files by repeating the `-c` flag. You can use this, for example, for setting defaults in a base configuration file, and overwrite settings via local configuration files.

In addition to overwriting settings using multiple configuration files, individual settings can be overwritten using `-E <setting>=<value>`. The `<value>` can be either a single value or a complex object, such as a list or dictionary.

For example, given the following configuration:

```yaml
output.elasticsearch:
  hosts: ["http://localhost:9200"]
  username: username
  password: password
```

You can disable the Elasticsearch output and write all events to the console by setting:

```sh
-E output='{elasticsearch.enabled: false, console.pretty: true}'
```

Any complex objects that you specify at the command line are merged with the original configuration, and the following configuration is passed to the Beat:

```yaml
output.elasticsearch:
  enabled: false
  hosts: ["http://localhost:9200"]
  username: username
  password: password

output.console:
  pretty: true
```

