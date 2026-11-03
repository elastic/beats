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

By default, Packetbeat resolves a connection to a process by reading the list of active TCP and UDP connections and matching the connection's socket against the file descriptors held by the running processes. This information is available via system interfaces: the `/proc` file system in Linux and the IP Helper API (`iphlpapi.dll`) on Windows, so Packetbeat doesn't need a kernel module. Because the socket must still exist when the lookup runs, short-lived connections can be missed; on Linux the `kernel_tracing` backend avoids this limitation.

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


### `backend` [_procs_backend]

```{applies_to}
stack: ga 9.7.0+
```

The mechanism used to resolve connections to processes. The following values are supported:

* `procfs` (default): Look the socket up in the OS socket table (`/proc` on Linux, the IP Helper API on Windows). Short-lived connections whose socket already closed at lookup time cannot be resolved.
* `kernel_tracing`: Track socket and process lifecycle with [quark](https://github.com/elastic/quark) using eBPF, so TCP connections are resolved reliably even after their socket closed. This backend is only available on Linux amd64/arm64. Packetbeat fails to start when this backend is requested but the kernel tracer cannot be started.
* `auto`: The same as `kernel_tracing` when the kernel tracer can be started. Otherwise Packetbeat logs a warning and runs with `procfs` instead of failing to start.

With both `kernel_tracing` and `auto`, UDP connections and TCP connections that the kernel tracer has not seen yet are still looked up with the `procfs` mechanism.

Example configuration:

```yaml
packetbeat.procs.enabled: true
packetbeat.procs.backend: auto
```

#### Requirements for `kernel_tracing` [_procs_kernel_tracing_requirements]

The `kernel_tracing` backend loads eBPF programs into the kernel, which needs more than the packet capture privileges Packetbeat otherwise requires:

* Linux amd64 or arm64 with a kernel that supports BPF CO-RE (BTF) and BPF ring buffers, generally 5.10 or later. Some distribution kernels ship without BTF, in which case the backend is unavailable.
* Run as root, or with the `CAP_BPF`, `CAP_PERFMON` and `CAP_SYS_RESOURCE` capabilities in addition to the capabilities needed for packet capture. On kernels older than 5.8, `CAP_SYS_ADMIN` is required instead of `CAP_BPF` and `CAP_PERFMON`.
* Read access to `tracefs` (normally mounted at `/sys/kernel/tracing` or `/sys/kernel/debug/tracing`) and, for process metadata, read access to `/proc`.
* When running in a container, the container must share the host PID namespace and have the capabilities above. The backend is not available under Elastic Agent's unprivileged mode.
* Packetbeat's default [seccomp policy](/reference/packetbeat/linux-seccomp.md) is extended to allow the system calls the kernel tracer needs (`bpf`, `perf_event_open`, `memfd_create`). If you configure a custom `seccomp` policy, it must allow them too.

The backend keeps closed connections resolvable for the flow timeout plus the flow reporting period, plus a margin, so that flows reported after a connection ended are still attributed to a process. Long flow timeouts therefore increase the memory kept by the kernel tracer.

The kernel tracer delivers a new connection to Packetbeat a fraction of a second after it is established. A protocol transaction that completes before that, on a connection that lasts only a few tens of milliseconds, may still be published without process information. The flow events for that connection are reported later and are enriched.


### `shutdown_timeout` [shutdown-timeout]

How long Packetbeat waits on shutdown. By default, this option is disabled. Packetbeat will wait for `shutdown_timeout` and then close. It will not track if all events were sent previously.

Example configuration:

```yaml
packetbeat.shutdown_timeout: 5s
```


### `overwrite_pipelines` [_overwrite_pipelines]

By default Ingest pipelines are not updated if a pipeline with the same ID already exists. If this option is enabled Packetbeat overwrites pipelines every time a new Elasticsearch connection is established.

The default value is `false`.

