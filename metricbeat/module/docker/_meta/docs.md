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
