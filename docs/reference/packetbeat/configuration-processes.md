---
navigation_title: "Processes"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-processes.html
---

# Configure which processes to monitor [configuration-processes]


This section of the `packetbeat.yml` config file is optional, but configuring the processes enables Packetbeat to show you not only the servers that the traffic is flowing between, but also the processes. Packetbeat can even show you the traffic between two processes running on the same host, which is particularly useful when you have many services running on the same server. By default, process enrichment is disabled.

When Packetbeat starts, and then periodically afterwards, it scans the process table for processes that match the configuration file. For each of these processes, it monitors which file descriptors it has opened. When a new packet is captured, it reads the list of active TCP and UDP connections and matches the corresponding one with the list of file descriptors.

All this information is available via system interfaces: The `/proc` file system in Linux and the IP Helper API (`iphlpapi.dll`) on Windows, so Packetbeat doesn’t need a kernel module.

::::{note}
Process monitoring is currently only supported on Linux and Windows systems. Packetbeat automatically disables process monitoring when it detects other operating systems.
::::


Example configuration:

```yaml
packetbeat.procs.enabled: true
```

When the process monitor is enabled, it will enrich all the events whose source or destination is a local process. The `source.process` and/or `destination.process` fields will be added to an event, when the server side or client side of the connection belong to a local process, respectively.


## Configuration options [_configuration_options_14]

You can specify the following process monitoring options in the `monitored` section of the `packetbeat.yml` config file to customize the name of process:


### `process` [_process]

The name of the process as it will appear in the published transactions. The name doesn’t have to match the name of the executable, so feel free to choose something more descriptive (for example,  "myapp" instead of "gunicorn").


### `cmdline_grep` [_cmdline_grep]

The name used to identify the process at run time. When Packetbeat starts, and then periodically afterwards, it scans the process table for processes that match the values specified for this option. The match is done against the process' command line as read from `/proc/<pid>/cmdline`.


### `shutdown_timeout` [shutdown-timeout]

How long Packetbeat waits on shutdown. By default, this option is disabled. Packetbeat will wait for `shutdown_timeout` and then close. It will not track if all events were sent previously.

Example configuration:

```yaml
packetbeat.shutdown_timeout: 5s
```


### `overwrite_pipelines` [_overwrite_pipelines]

By default Ingest pipelines are not updated if a pipeline with the same ID already exists. If this option is enabled Packetbeat overwrites pipelines every time a new Elasticsearch connection is established.

The default value is `false`.

