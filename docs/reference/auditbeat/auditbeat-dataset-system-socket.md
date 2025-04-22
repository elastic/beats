---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-dataset-system-socket.html
---

# System socket dataset [auditbeat-dataset-system-socket]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the `socket` dataset of the system module. It allows to monitor network traffic to and from running processes. It’s main features are:

* Supports TCP and UDP sockets over IPv4 and IPv6.
* Outputs per-flow bytes and packets counters.
* Enriches the flows with [process](ecs://reference/ecs-process.md) and [user](ecs://reference/ecs-user.md) information.
* Provides information similar to Packetbeat’s flow monitoring with reduced CPU and memory usage.
* Works on stock kernels without the need of custom modules, external libraries or development headers.
* Correlates IP addresses with DNS requests.

This dataset does not analyze application-layer protocols nor provide any other advanced features present in Packetbeat: - Monitor network traffic whose destination is not a local process, as is the case with traffic forwarding. - Monitor layer 2 traffic, ICMP or raw sockets.


## Implementation [_implementation_2]

It is implemented for Linux only and currently supports x86 (32 and 64 bit) architectures.

The dataset uses [KProbe-based event tracing](https://www.kernel.org/doc/Documentation/trace/kprobetrace.txt) to monitor TCP and UDP sockets over IPv4 and IPv6, providing flow monitoring that includes byte and packet counters, as well as the local process and user involved in the flow. It does so by plugin into the TCP/IP stack to generate custom tracing events avoiding the need to copy network traffic to user space.

By not relying on periodic polling, this approach enables the dataset to perform near real-time monitoring of the system without the risk of missing short lived connections or processes.


## Requirements [_requirements]

Features used by the `socket` dataset require a minimum Linux kernel version of 3.12 (vanilla). However, some distributions have backported those features to older kernels. The following (non-exhaustive) lists the different distributions under which the dataset is known to work:

| Distribution | kernel version | Works? |
| --- | --- | --- |
| CentOS 6.5 | 2.6.32-431.el6 | NO[[1]](#anchor-1) |
| CentOS 6.9 | 2.6.32-696.30.1.el6 | ✓ |
| CentOS 7.6 | 3.10.0-957.1.3.el7 | ✓ |
| RHEL 8 | 4.18.0-80.rhel8 | ✓ |
| Debian 8 | 3.16.0-6 | ✓ |
| Debian 9 | 4.9.0-8 | ✓ |
| Debian 10 | 4.19.0-5 | ✓ |
| SLES 12 | 4.4.73-5 | ✓ |
| Ubuntu 12.04 | 3.2.0-126 | NO[[1]](#anchor-1) |
| Ubuntu 14.04.6 | 3.13.0-170 | ✓ |
| Ubuntu 16.04.3 | 4.4.0-97 | ✓ |
| AWS Linux 2 | 4.14.138-114.102 | ✓ |

$$$anchor-1$$$
[[1]](#anchor-1): These systems lack [PERF_EVENT_IOC_ID ioctl.](https://lore.kernel.org/patchwork/patch/399251/) Support might be added in a future release.

The dataset needs CAP_SYS_ADMIN and CAP_NET_ADMIN in order to work.


### Kernel configuration [_kernel_configuration]

A kernel built with the following configuration options enabled is required:

* `CONFIG_KPROBE_EVENTS`: Enables the KProbes subsystem.
* `CONFIG_DEBUG_FS`: For kernels laking `tracefs` support (<4.1).
* `CONFIG_IPV6`: IPv6 support in the kernel is needed even if disabled with `socket.enable_ipv6: false`.

These settings are enabled by default in most distributions.

The following configuration settings can prevent the dataset from starting:

* `/sys/kernel/debug/kprobes/enabled` must be 1.
* `/proc/sys/net/ipv6/conf/lo/disable_ipv6` (IPv6 enabled in loopback device) is required when running with IPv6 enabled.


### Running on docker [_running_on_docker]

The dataset can monitor the Docker host when running inside a container. However it needs to run on a `privileged` container with `CAP_NET_ADMIN`. The docker container running Auditbeat needs access to the host’s tracefs or debugfs directory. This is achieved by bind-mounting `/sys`.


## Configuration [_configuration_2]

The following options are available for the `socket` dataset:

* `socket.tracefs_path` (default: none)

Must point to the mount-point of `tracefs` or the `tracing` directory inside `debugfs`. If this option is not specified, Auditbeat will look for the default locations: `/sys/kernel/tracing` and `/sys/kernel/debug/tracing`. If not found, it will attempt to mount `tracefs` and `debugfs` at their default locations.

* `socket.enable_ipv6` (default: unset)

Determines whether IPv6 must be monitored. When unset (default), IPv6 support is automatically detected. Even when IPv6 is disabled, in order to run the dataset you still need a kernel with IPv6 support (the `ipv6` module must be loaded if compiled as a module).

* `socket.flow_inactive_timeout` (default: 30s)

Determines how long a flow has to be inactive to be considered closed.

* `socket.flow_termination_timeout` (default: 5s)

Determines how long to wait after a socket has been closed for out of order packets. With TCP, some packets can be received shortly after a socket is closed. If set too low, additional flows will be generated for those packets.

* `socket.socket_inactive_timeout` (default: 1m)

How long a socket can be inactive to be evicted from the internal cache. A lower value reduces memory usage at the expense of some flows being reported as multiple partial flows.

* `socket.perf_queue_size` (default: 4096)

The number of tracing samples that can be queued for processing. A larger value uses more memory but reduces the chances of samples being lost when the system is under heavy load.

* `socket.lost_queue_size` (default: 128)

The number of lost samples notifications that can be queued.

* `socket.ring_size_exponent` (default: 7)

Controls the number of memory pages allocated for the per-CPU ring-buffer used to receive samples from the kernel. The actual amount of memory used is Number_of_CPUs x Page_Size(4KB) x 2ring_size_exponent. That is 0.5 MiB of RAM per CPU with the default value.

* `socket.clock_max_drift` (default: 100ms)

Defines the maximum difference between the kernel internal clock and the reference time used to timestamp events.

* `socket.clock_sync_period` (default: 10s)

Controls how often clock synchronization events are generated to measure drift between the kernel clock and the dataset’s reference clock.

* `socket.guess_timeout` (default: 15s)

The maximum time an individual guess is allowed to run.

* `socket.dns.enabled` (default: true)

If DNS traffic must be monitored to enrich network flows with DNS information.

* `socket.dns.type` (default: af_packet)

The method used to monitor DNS traffic. Currently, only `af_packet` is supported.

* `socket.dns.af_packet.interface` (default: any)

The network interface where DNS will be monitored.

* `socket.dns.af_packet.snaplen` (default: 1024)

Maximum number of bytes to copy for each captured packet.

## Fields [_fields_7]

For a description of each field in the dataset, see the [exported fields](/reference/auditbeat/exported-fields-system.md) section.

Here is an example document generated by this dataset:

```json
{
    "@timestamp":"2019-08-22T20:46:40.173Z",
    "@metadata":{
        "beat":"auditbeat",
        "type":"_doc",
        "version":"7.4.0"
    },
    "server":{
        "ip":"151.101.66.217",
        "port":80,
        "packets":5,
        "bytes":437
    },
    "user":{
        "name":"vagrant",
        "id":"1000"
    },
    "network":{
        "packets":10,
        "bytes":731,
        "community_id":"1:jdjL1TkdpF1v1GM0+JxRRp+V7KI=",
        "direction":"outbound",
        "type":"ipv4",
        "transport":"tcp"
    },
    "group":{
        "id":"1000",
        "name":"vagrant"
    },
    "client":{
        "ip":"10.0.2.15",
        "port":40192,
        "packets":5,
        "bytes":294
    },
    "event":{
        "duration":30728600,
        "module":"system",
        "dataset":"socket",
        "kind":"event",
        "action":"network_flow",
        "category":"network",
        "start":"2019-08-22T20:46:35.001Z",
        "end":"2019-08-22T20:46:35.032Z"
    },
    "ecs":{
        "version":"1.0.1"
    },
    "host":{
        "name":"stretch",
        "containerized":false,
        "hostname":"stretch",
        "architecture":"x86_64",
        "os":{
            "name":"Debian GNU/Linux",
            "kernel":"4.9.0-8-amd64",
            "codename":"stretch",
            "platform":"debian",
            "version":"9 (stretch)",
            "family":"debian"
        },
        "id":"b3531219b5b4449eadbec59d47945649"
    },
    "agent":{
        "version":"7.4.0",
        "type":"auditbeat",
        "ephemeral_id":"f7b0ab1a-da9e-4525-9252-59ecb68139f8",
        "hostname":"stretch",
        "id":"88862e07-b13a-4166-b1ef-b3e55b4a0cf2"
    },
    "process":{
        "pid":4970,
        "name":"curl",
        "args":[
            "curl",
            "http://elastic.co/",
            "-o",
            "/dev/null"
        ],
        "executable":"/usr/bin/curl",
        "created":"2019-08-22T20:46:34.928Z"
    },
    "system":{
        "audit":{
            "socket":{
                "kernel_sock_address":"0xffff8de29d337000",
                "internal_version":"1.0.3",
                "uid":1000,
                "gid":1000,
                "euid":1000,
                "egid":1000
            }
        }
    },
    "destination":{
        "ip":"151.101.66.217",
        "port":80,
        "packets":5,
        "bytes":437
    },
    "source":{
        "port":40192,
        "packets":5,
        "bytes":294,
        "ip":"10.0.2.15"
    },
    "flow":{
        "final":true,
        "complete":true
    },
    "service":{
        "type":"system"
    }
}
```


