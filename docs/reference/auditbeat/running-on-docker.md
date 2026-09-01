---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/running-on-docker.html
applies_to:
  stack: ga
  serverless: ga
---

# Run Auditbeat on Docker [running-on-docker]

Docker images for Auditbeat are available from the Elastic Docker registry. The base image is [Red Hat Universal Base Image 10 Minimal](https://hub.docker.com/r/redhat/ubi10-minimal).

A list of all published Docker images and tags is available at [www.docker.elastic.co](https://www.docker.elastic.co).

These images are free to use under the Elastic license. They contain open source and free commercial features and access to paid commercial features. [Start a 30-day trial](docs-content://deploy-manage/license/manage-your-license-in-self-managed-cluster.md) to try out all of the paid commercial features. See the [Subscriptions](https://www.elastic.co/subscriptions) page for information about Elastic license levels.

## Pull the image [_pull_the_image]

Obtaining Auditbeat for Docker is as simple as issuing a `docker pull` command against the Elastic Docker registry.

% ::::{warning} subs=true
% Version {{version.stack}} of Auditbeat has not yet been released. No Docker image is currently available for Auditbeat {{version.stack}}.
% ::::


```sh subs=true
docker pull docker.elastic.co/beats/auditbeat:{{version.stack}}
```

Alternatively, you can download other Docker images that contain only features available under the Apache 2.0 license. To download the images, go to [www.docker.elastic.co](https://www.docker.elastic.co).

As another option, you can use the hardened [Wolfi](https://wolfi.dev/) image. Using Wolfi images requires Docker version 20.10.10 or higher. For details about why the Wolfi images have been introduced, refer to our article [Reducing CVEs in Elastic container images](https://www.elastic.co/blog/reducing-cves-in-elastic-container-images).

```bash subs=true
docker pull docker.elastic.co/beats/auditbeat-wolfi:{{version.stack}}
```


## Optional: Verify the image [_optional_verify_the_image]

You can use the [Cosign application](https://docs.sigstore.dev/cosign/installation/) to verify the Auditbeat Docker image signature.

% ::::{warning} subs=true
% Version {{version.stack}} of Auditbeat has not yet been released. No Docker image is currently available for Auditbeat {{version.stack}}.
% ::::


```sh subs=true
wget https://artifacts.elastic.co/cosign.pub
cosign verify --key cosign.pub docker.elastic.co/beats/auditbeat:{{version.stack}}
```

The `cosign` command prints the check results and the signature payload in JSON format:

```sh subs=true
Verification for docker.elastic.co/beats/auditbeat:{{version.stack}} --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The signatures were verified against the specified public key
```


## Run the Auditbeat setup [_run_the_auditbeat_setup]

::::{important}
A [known issue](https://github.com/elastic/beats/issues/42038) in version 8.17.0 prevents {{beats}} Docker images from starting when no options are provided. When running an image on that version, add an `--environment container` parameter to avoid the problem. This is planned to be addressed in issue [#42060](https://github.com/elastic/beats/pull/42060).
::::


Running Auditbeat with the setup command will create the index pattern and load visualizations , dashboards, and machine learning jobs.  Run this command:

```sh subs=true
docker run --rm \
  --cap-add="AUDIT_CONTROL" \
  --cap-add="AUDIT_READ" \
  docker.elastic.co/beats/auditbeat:{{version.stack}} \
  setup -E setup.kibana.host=kibana:5601 \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1>
```
1. Substitute your {{kib}} and {{es}} hosts and ports. 

   If you are using {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using this syntax:

   ```sh
   -E cloud.id=<Cloud ID from Elastic Cloud Hosted> \
   -E cloud.auth=elastic:<elastic password>
   ```


## Run Auditbeat on a read-only file system [_run_auditbeat_on_a_read_only_file_system]

If you’d like to run Auditbeat in a Docker container on a read-only file system, you can do so by specifying the `--read-only` option. Auditbeat requires a stateful directory to store application data, so with the `--read-only` option you also need to use the `--mount` option to specify a path to where that data can be stored.

For example:

```sh subs=true
docker run --rm \
  --mount type=bind,source=$(pwd)/data,destination=/usr/share/auditbeat/data \
  --read-only \
  docker.elastic.co/beats/auditbeat:{{version.stack}}
```


## Configure Auditbeat on Docker [_configure_auditbeat_on_docker]

The Docker image provides several methods for configuring Auditbeat. The conventional approach is to provide a configuration file via a volume mount, but it’s also possible to create a custom image with your configuration included.

### Example configuration file [_example_configuration_file]

Download this example configuration file as a starting point:

```sh subs=true
curl -L -O https://raw.githubusercontent.com/elastic/beats/{{ version.stack | M.M }}/deploy/docker/auditbeat.docker.yml
```


### Volume-mounted configuration [_volume_mounted_configuration]

One way to configure Auditbeat on Docker is to provide `auditbeat.docker.yml` via a volume mount. With `docker run`, the volume mount can be specified like this.

```sh subs=true
docker run -d \
  --name=auditbeat \
  --user=root \
  --volume="$(pwd)/auditbeat.docker.yml:/usr/share/auditbeat/auditbeat.yml:ro" \
  --cap-add="AUDIT_CONTROL" \
  --cap-add="AUDIT_READ" \
  --pid=host \
  docker.elastic.co/beats/auditbeat:{{version.stack}} -e \
  --strict.perms=false \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1>
```
1. Substitute your {{es}} hosts and ports.

   If you are using {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using the syntax shown earlier.


### Customize your configuration [_customize_your_configuration]

The `auditbeat.docker.yml` downloaded earlier should be customized for your environment. See [Configure](/reference/auditbeat/configuring-howto-auditbeat.md) for more details. Edit the configuration file and customize it to match your environment then re-deploy your Auditbeat container.


### Custom image configuration [_custom_image_configuration]

It’s possible to embed your Auditbeat configuration in a custom image. Here is an example Dockerfile to achieve this:

```dockerfile subs=true
FROM docker.elastic.co/beats/auditbeat:{{version.stack}}
COPY auditbeat.yml /usr/share/auditbeat/auditbeat.yml
```



## Special requirements [_special_requirements]

Auditbeat modules and datasets have different privilege requirements. The table below shows the minimum flags needed for each component when running on Docker. Grant only the capabilities your configuration actually uses.

| Component | `--cap-add` flags | `--pid=host` | Volume mounts |
|---|---|---|---|
| **auditd** (multicast, default) | `AUDIT_READ` | No | — |
| **auditd** (unicast) | `AUDIT_CONTROL`, `AUDIT_READ` | **Yes** | — |
| **file_integrity** (fsnotify, default) | None | No | Paths to monitor (read-only) |
| **file_integrity** (kprobes) | `SYS_ADMIN` | No | `-v /sys:/sys` |
| **file_integrity** (ebpf) | `SYS_ADMIN`, `BPF` | No | `-v /sys:/sys` |
| **system/host** | None | No | — |
| **system/login** | None (utmp group membership) | No | `-v /var/log:/var/log:ro` |
| **system/package** (dpkg) | None | No | `-v /var/lib/dpkg:/var/lib/dpkg:ro` |
| **system/package** (RPM) | None (or root for RPM DB) | No | `-v /var/lib/rpm:/var/lib/rpm:ro` |
| **system/process** | `SYS_PTRACE` (recommended) | **Yes** | — |
| **system/socket** | `SYS_ADMIN`, `NET_ADMIN` | No | `-v /sys:/sys` |
| **system/user** (default) | None | No | — |
| **system/user** (`detect_password_changes: true`) | None (shadow group) | No | `-v /etc/shadow:/etc/shadow:ro` |
| **add_session_metadata** (procfs) | `SYS_PTRACE` | **Yes** | — |
| **add_session_metadata** (kernel_tracing/kprobes) | `SYS_ADMIN` | **Yes** | `-v /sys/kernel/debug:/sys/kernel/debug` |
| **add_session_metadata** (kernel_tracing/eBPF) | `SYS_ADMIN`, `BPF` | **Yes** | `-v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/bpf:/sys/fs/bpf` |

For more detail on what each component requires and why, see the individual module and dataset pages:

* [Auditd module](/reference/auditbeat/auditbeat-module-auditd.md)
* [File Integrity module](/reference/auditbeat/auditbeat-module-file_integrity.md)
* [System module](/reference/auditbeat/auditbeat-module-system.md)
* [Add session metadata processor](/reference/auditbeat/add-session-metadata.md)

**Typical full-featured setup**

The volume-mounted configuration example earlier in this page uses `--cap-add=AUDIT_CONTROL --cap-add=AUDIT_READ --pid=host` — the minimum needed for the `auditd` module in unicast mode. If you also enable the `system/socket` dataset or `add_session_metadata` with kernel tracing, add the additional flags from the table above.

```sh subs=true
docker run --cap-add=AUDIT_CONTROL --cap-add=AUDIT_READ --user=root --pid=host docker.elastic.co/beats/auditbeat:{{version.stack}}
```


