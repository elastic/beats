---
navigation_title: "Processes"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/configuration-processes.html
applies_to:
  stack: ga
  serverless: ga
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


### `max_proc_read_freq` [_max_proc_read_freq]

```{applies_to}
stack: ga 9.5+
```

The minimum time between full scans of process sockets, kept separately for TCP and UDP. On Linux, a full scan reads the file descriptors of every process on the host, so on hosts with many processes it can use a lot of CPU. When a lookup finds no process for a local endpoint, Packetbeat only scans again if at least this long has passed since the previous scan finished. Until then, lookups that miss return no process.

Protocol transactions look up the process once, so a transaction on a connection that is opened and closed between two scans is published without process information. Flow events are looked up again each time they are reported.

On Linux, Packetbeat first reads the socket table, which is much cheaper, and skips the scan when no open socket uses the port, because the scan could not find it.

After each lookup of the socket table, whether or not it led to a scan, Packetbeat waits nine times as long as that lookup took, including reading the details of new processes, before the next one. Lookups for each transport therefore take about a tenth of the time. TCP and UDP lookups cannot run at the same time, so a long scan delays enrichment for both. On hosts where a scan takes a few milliseconds, almost every miss can still scan. On hosts where a scan is slow, the wait grows with it. This wait always applies unless the option is negative.

The default, `0`, adds no fixed minimum. A positive value also keeps full scans at least that far apart, to reduce CPU use further at the cost of missing more new connections. A negative value, such as `-1s`, disables both limits, so every miss triggers a scan. When several streams set this option, the largest value is used, unless any of them is negative, in which case both limits are disabled.

A process found by a scan stays associated with its endpoint for at least 10 seconds, or for the wait before the next scan plus a few seconds if that is longer, so long-lived services stay enriched between scans. The association is not checked again while it lasts, so a longer interval also lengthens the time during which events can show the previous process if another process reuses the same port.

Earlier releases, including the patch releases of 9.5 and 9.6 before this option took effect, accept it but ignore it.

Example configuration:

```yaml
packetbeat.procs.enabled: true
packetbeat.procs.max_proc_read_freq: 5s
```


### `shutdown_timeout` [shutdown-timeout]

How long Packetbeat waits on shutdown. By default, this option is disabled. Packetbeat will wait for `shutdown_timeout` and then close. It will not track if all events were sent previously.

Example configuration:

```yaml
packetbeat.shutdown_timeout: 5s
```


### `overwrite_pipelines` [_overwrite_pipelines]

By default Ingest pipelines are not updated if a pipeline with the same ID already exists. If this option is enabled Packetbeat overwrites pipelines every time a new Elasticsearch connection is established.

The default value is `false`.

