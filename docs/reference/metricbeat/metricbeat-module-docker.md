---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-docker.html
---

# Docker module [metricbeat-module-docker]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/docker/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module fetches metrics from [Docker](https://www.docker.com/) containers. The default metricsets are: `container`, `cpu`, `diskio`, `healthcheck`, `info`, `memory` and `network`. The `image` metricset is not enabled by default.


## Compatibility [_compatibility_16]

The Docker module is currently tested on Linux and Mac with the community edition engine, versions 1.11 and 17.09.0-ce. It is not tested on Windows, but it should also work there.

The Docker module supports collection of metrics from Podman’s Docker-compatible API on Metricbeat 8.16.2 and 8.17.1, and higher versions. It has been tested on Linux and Mac with Podman Rest API v2.0.0 and above.


## Module-specific configuration notes [_module_specific_configuration_notes_6]

It is strongly recommended that you run Docker metricsets with a [`period`](/reference/metricbeat/configuration-metricbeat.md#metricset-period) that is 3 seconds or longer. The request to the Docker API already takes up to 2 seconds. Specifying less than 3 seconds will result in requests that timeout, and no data will be reported for those requests. In the case of Podman, the configuration parameter `podman` should be set to `true`. This enables streaming of container stats output, which allows for more accurate CPU percentage calculations when using Podman.


## Example configuration [_example_configuration_18]

The Docker module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: docker
  metricsets:
    - "container"
    - "cpu"
    - "diskio"
    - "event"
    - "healthcheck"
    - "info"
    #- "image"
    - "memory"
    - "network"
    #- "network_summary"
  hosts: ["unix:///var/run/docker.sock"]
  period: 10s
  enabled: true

  # If set to true, replace dots in labels with `_`.
  #labels.dedot: false

  # Docker module supports metrics collection from podman's docker compatible API. In case of podman set to true.
  # podman: false

  # Skip metrics for certain device major numbers in docker/diskio.
  # Necessary on systems with software RAID, device mappers,
  # or other configurations where virtual disks will sum metrics from other disks.
  # By default, it will skip devices with major numbers 9 or 253.
  #skip_major: []

  # If set to true, collects metrics per core.
  #cpu.cores: true

  # To connect to Docker over TLS you must specify a client and CA certificate.
  #ssl:
    #certificate_authority: "/etc/pki/root/ca.pem"
    #certificate:           "/etc/pki/client/cert.pem"
    #key:                   "/etc/pki/client/cert.key"
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md).


## Metricsets [_metricsets_24]

The following metricsets are available:

* [container](/reference/metricbeat/metricbeat-metricset-docker-container.md)
* [cpu](/reference/metricbeat/metricbeat-metricset-docker-cpu.md)
* [diskio](/reference/metricbeat/metricbeat-metricset-docker-diskio.md)
* [event](/reference/metricbeat/metricbeat-metricset-docker-event.md)
* [healthcheck](/reference/metricbeat/metricbeat-metricset-docker-healthcheck.md)
* [image](/reference/metricbeat/metricbeat-metricset-docker-image.md)
* [info](/reference/metricbeat/metricbeat-metricset-docker-info.md)
* [memory](/reference/metricbeat/metricbeat-metricset-docker-memory.md)
* [network](/reference/metricbeat/metricbeat-metricset-docker-network.md)
* [network_summary](/reference/metricbeat/metricbeat-metricset-docker-network_summary.md)











