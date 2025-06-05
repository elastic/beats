---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/running-on-docker.html
---

# Run Metricbeat on Docker [running-on-docker]

Docker images for Metricbeat are available from the Elastic Docker registry. The base image is [centos:7](https://hub.docker.com/_/centos/).

A list of all published Docker images and tags is available at [www.docker.elastic.co](https://www.docker.elastic.co).

These images are free to use under the Elastic license. They contain open source and free commercial features and access to paid commercial features. [Start a 30-day trial](docs-content://deploy-manage/license/manage-your-license-in-self-managed-cluster.md) to try out all of the paid commercial features. See the [Subscriptions](https://www.elastic.co/subscriptions) page for information about Elastic license levels.

## Pull the image [_pull_the_image]

Obtaining Metricbeat for Docker is as simple as issuing a `docker pull` command against the Elastic Docker registry.

% ::::{warning} subs=true
% Version {{stack-version}} of Metricbeat has not yet been released. No Docker image is currently available for Metricbeat {{stack-version}}.
% ::::


```sh subs=true
docker pull docker.elastic.co/beats/metricbeat:{{stack-version}}
```

Alternatively, you can download other Docker images that contain only features available under the Apache 2.0 license. To download the images, go to [www.docker.elastic.co](https://www.docker.elastic.co).

As another option, you can use the hardened [Wolfi](https://wolfi.dev/) image. Using Wolfi images requires Docker version 20.10.10 or higher. For details about why the Wolfi images have been introduced, refer to our article [Reducing CVEs in Elastic container images](https://www.elastic.co/blog/reducing-cves-in-elastic-container-images).

```bash subs=true
docker pull docker.elastic.co/beats/metricbeat-wolfi:{{stack-version}}
```


## Optional: Verify the image [_optional_verify_the_image]

You can use the [Cosign application](https://docs.sigstore.dev/cosign/installation/) to verify the Metricbeat Docker image signature.

% ::::{warning}
% Version {{stack-version}} of Metricbeat has not yet been released. No Docker image is currently available for Metricbeat {{stack-version}}.
% ::::


```sh subs=true
wget https://artifacts.elastic.co/cosign.pub
cosign verify --key cosign.pub docker.elastic.co/beats/metricbeat:{{stack-version}}
```

The `cosign` command prints the check results and the signature payload in JSON format:

```sh subs=true
Verification for docker.elastic.co/beats/metricbeat:{{stack-version}} --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The signatures were verified against the specified public key
```


## Run the Metricbeat setup [_run_the_metricbeat_setup]

::::{important}
A [known issue](https://github.com/elastic/beats/issues/42038) in version 8.17.0 prevents {{beats}} Docker images from starting when no options are provided. When running an image on that version, add an `--environment container` parameter to avoid the problem. This is planned to be addressed in issue [#42060](https://github.com/elastic/beats/pull/42060).
::::


Running Metricbeat with the setup command will create the index pattern and load visualizations , dashboards, and machine learning jobs.  Run this command:

```sh subs=true
docker run --rm \
docker.elastic.co/beats/metricbeat:{{stack-version}} \
setup -E setup.kibana.host=kibana:5601 \
-E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Kibana and Elasticsearch hosts and ports.
2. If you are using the {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using this syntax:


```shell
-E cloud.id=<Cloud ID from Elastic Cloud Hosted> \
-E cloud.auth=elastic:<elastic password>
```


## Run Metricbeat on a read-only file system [_run_metricbeat_on_a_read_only_file_system]

If you’d like to run Metricbeat in a Docker container on a read-only file system, you can do so by specifying the `--read-only` option. Metricbeat requires a stateful directory to store application data, so with the `--read-only` option you also need to use the `--mount` option to specify a path to where that data can be stored.

For example:

```sh subs=true
docker run --rm \
  --mount type=bind,source=$(pwd)/data,destination=/usr/share/metricbeat/data \
  --read-only \
  docker.elastic.co/beats/metricbeat:{{stack-version}}
```


## Configure Metricbeat on Docker [_configure_metricbeat_on_docker]

The Docker image provides several methods for configuring Metricbeat. The conventional approach is to provide a configuration file via a volume mount, but it’s also possible to create a custom image with your configuration included.

### Example configuration file [_example_configuration_file]

Download this example configuration file as a starting point:

```sh subs=true
curl -L -O https://raw.githubusercontent.com/elastic/beats/{{major-version}}/deploy/docker/metricbeat.docker.yml
```


### Volume-mounted configuration [_volume_mounted_configuration]

One way to configure Metricbeat on Docker is to provide `metricbeat.docker.yml` via a volume mount. With `docker run`, the volume mount can be specified like this.

```sh subs=true
docker run -d \
  --name=metricbeat \
  --user=root \
  --volume="$(pwd)/metricbeat.docker.yml:/usr/share/metricbeat/metricbeat.yml:ro" \
  --volume="/var/run/docker.sock:/var/run/docker.sock:ro" \
  --volume="/sys/fs/cgroup:/hostfs/sys/fs/cgroup:ro" \
  --volume="/proc:/hostfs/proc:ro" \
  --volume="/:/hostfs:ro" \
  docker.elastic.co/beats/metricbeat:{{stack-version}} metricbeat -e \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Elasticsearch hosts and ports.
2. If you are using the {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using the syntax shown earlier.



### Customize your configuration [_customize_your_configuration]

The `metricbeat.docker.yml` file you downloaded earlier is configured to deploy Beats modules based on the Docker labels applied to your containers.  See [Hints based autodiscover](/reference/metricbeat/configuration-autodiscover-hints.md) for more details. Add labels to your application Docker containers, and they will be picked up by the Beats autodiscover feature when they are deployed.  Here is an example command for an Apache HTTP Server container with labels to configure the Filebeat and Metricbeat modules for the Apache HTTP Server:

```sh
docker run \
  --label co.elastic.logs/module=apache2 \
  --label co.elastic.logs/fileset.stdout=access \
  --label co.elastic.logs/fileset.stderr=error \
  --label co.elastic.metrics/module=apache \
  --label co.elastic.metrics/metricsets=status \
  --label co.elastic.metrics/hosts='${data.host}:${data.port}' \
  --detach=true \
  --name my-apache-app \
  -p 8080:80 \
  httpd:2.4
```


### Custom image configuration [_custom_image_configuration]

It’s possible to embed your Metricbeat configuration in a custom image. Here is an example Dockerfile to achieve this:

```dockerfile subs=true
FROM docker.elastic.co/beats/metricbeat:{{stack-version}}
COPY --chown=root:metricbeat metricbeat.yml /usr/share/metricbeat/metricbeat.yml
```


### Monitor the host machine [monitoring-host]

When executing Metricbeat in a container, there are some important things to be aware of if you want to monitor the host machine or other containers. Let’s walk-through some examples using Docker as our container orchestration tool.

This example highlights the changes required to make the system module work properly inside of a container. This enables Metricbeat to monitor the host machine from within the container.

```sh subs=true
docker run \
  --mount type=bind,source=/proc,target=/hostfs/proc,readonly \ <1>
  --mount type=bind,source=/sys/fs/cgroup,target=/hostfs/sys/fs/cgroup,readonly \ <2>
  --mount type=bind,source=/,target=/hostfs,readonly \ <3>
  --mount type=bind,source=/var/run/dbus/system_bus_socket,target=/hostfs/var/run/dbus/system_bus_socket,readonly \ <4>
  --env DBUS_SYSTEM_BUS_ADDRESS='unix:path=/hostfs/var/run/dbus/system_bus_socket' \ <4>
  --net=host \ <5>
  --cgroupns=host \ <6>
  docker.elastic.co/beats/metricbeat:{{stack-version}} -e --system.hostfs=/hostfs
```

1. Metricbeat’s [system module](/reference/metricbeat/metricbeat-module-system.md) collects much of its data through the Linux proc filesystem, which is normally located at `/proc`. Because containers are isolated as much as possible from the host, the data inside of the container’s `/proc` is different than the host’s `/proc`. To account for this, you can mount the host’s `/proc` filesystem inside of the container and tell Metricbeat to look inside the `/hostfs` directory when looking for `/proc` by using the `hostfs=/hostfs` config value.
2. By default, cgroup reporting is enabled for the [system process metricset](/reference/metricbeat/metricbeat-metricset-system-process.md), so you need to mount the host’s cgroup mountpoints within the container. They need to be mounted inside the directory specified by the `hostfs` config value.
3. If you want to be able to monitor filesystems from the host by using the [system filesystem metricset](/reference/metricbeat/metricbeat-metricset-system-filesystem.md), then those filesystems need to be mounted inside of the container. They can be mounted at any location.
4. The [system users metricset](/reference/metricbeat/metricbeat-metricset-system-users.md) and [system service metricset](/reference/metricbeat/metricbeat-metricset-system-service.md) both require access to dbus. Mount the dbus socket and set the `DBUS_SYSTEM_BUS_ADDRESS` environment variable to the mounted system socket path.
5. The [system network metricset](/reference/metricbeat/metricbeat-metricset-system-network.md) uses data from `/proc/net/dev`, or `/hostfs/proc/net/dev` when using `hostfs=/hostfs`. The only way to make this file contain the host’s network devices is to use the `--net=host` flag. This is due to Linux namespacing; simply bind mounting the host’s `/proc` to `/hostfs/proc` is not sufficient.
6. Runs the container using the host’s cgroup namespace, instead of a private namespace. While this is optional, [system process metricset](/reference/metricbeat/metricbeat-metricset-system-process.md) may produce more correct cgroup metrics when running in host mode.


::::{note}
The special filesystems `/proc` and `/sys` are only available if the host system is running Linux. Attempts to bind-mount these filesystems will fail on Windows and MacOS.
::::


If the [system socket metricset](/reference/metricbeat/metricbeat-metricset-system-socket.md) is being used on Linux, more privileges will need to be granted to Metricbeat. This metricset reads files from `/proc` that are an interface to internal objects owned by other users. The capabilities needed to read all these files (`sys_ptrace` and `dac_read_search`) are disabled by default on Docker. To grant these permissions these flags are needed too:

```sh
--user root --cap-add sys_ptrace --cap-add dac_read_search
```


### Monitor a service in another container [monitoring-service]

Next, let’s look at an example of monitoring a containerized service from a Metricbeat container.

```sh subs=true
docker run \
  --network=mysqlnet \ <1>
  -e MYSQL_PASSWORD=secret \ <2>
  docker.elastic.co/beats/metricbeat:{{stack-version}}
```

1. Placing the Metricbeat and MySQL containers on the same Docker network allows Metricbeat access to the exposed ports of the MySQL container, and makes the hostname `mysql` resolvable to Metricbeat.
2. If you do not want to hardcode certain values into your Metricbeat configuration, then you can pass them into the container either as environment variables or as command line flags to Metricbeat (see the `-E` CLI flag in [Command reference](/reference/metricbeat/command-line-options.md)).


The mysql module configuration would look like this:

```yaml
metricbeat.modules:
- module: mysql
  metricsets: ["status"]
  hosts: ["tcp(mysql:3306)/"] <1>
  username: root
  password: ${MYSQL_PASSWORD} <2>
```

1. The `mysql` hostname will resolve to the address of a container named `mysql` on the `mysqlnet` Docker network.
2. The `MYSQL_PASSWORD` variable will be evaluated at startup. If the variable is not set, this will lead to an error at startup.




