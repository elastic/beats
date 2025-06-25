::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/containerd/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


Containerd module collects cpu, memory and blkio statistics about running containers controlled by containerd runtime.

The current metricsets are: `cpu`, `blkio` and `memory` and are enabled by default.


## Prerequisites [_prerequisites]

`Containerd` daemon has to be configured to provide metrics before enabling containerd module.

In the configuration file located in `/etc/containerd/config.toml` metrics endpoint needs to be set and containerd daemon needs to be restarted.

```
[metrics]
    address = "127.0.0.1:1338"
```


## Compatibility [_compatibility_12]

The Containerd module is tested with the following versions of Containerd: v1.5.2


## Module-specific configuration notes [_module_specific_configuration_notes_5]

For cpu metricset if `calcpct.cpu` setting is set to true, cpu usage percentages will be calculated and more specifically fields `containerd.cpu.usage.total.pct`, `containerd.cpu.usage.kernel.pct`, `containerd.cpu.usage.user.pct`. Default value is true.

For memory metricset if `calcpct.memory` setting is set to true, memory usage percentages will be calculated and more specifically fields `containerd.memory.usage.pct` and  `containerd.memory.workingset.pct`. Default value is true.
