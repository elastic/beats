---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/running-on-docker.html
---

# Run Packetbeat on Docker [running-on-docker]

Docker images for Packetbeat are available from the Elastic Docker registry. The base image is [centos:7](https://hub.docker.com/_/centos/).

A list of all published Docker images and tags is available at [www.docker.elastic.co](https://www.docker.elastic.co).

These images are free to use under the Elastic license. They contain open source and free commercial features and access to paid commercial features. [Start a 30-day trial](docs-content://deploy-manage/license/manage-your-license-in-self-managed-cluster.md) to try out all of the paid commercial features. See the [Subscriptions](https://www.elastic.co/subscriptions) page for information about Elastic license levels.

## Pull the image [_pull_the_image]

Obtaining Packetbeat for Docker is as simple as issuing a `docker pull` command against the Elastic Docker registry.

% ::::{warning} subs=true
% Version {{stack-version}} of Packetbeat has not yet been released. No Docker image is currently available for Packetbeat {{stack-version}}.
% ::::


```sh subs=true
docker pull docker.elastic.co/beats/packetbeat:{{stack-version}}
```

Alternatively, you can download other Docker images that contain only features available under the Apache 2.0 license. To download the images, go to [www.docker.elastic.co](https://www.docker.elastic.co).

As another option, you can use the hardened [Wolfi](https://wolfi.dev/) image. Using Wolfi images requires Docker version 20.10.10 or higher. For details about why the Wolfi images have been introduced, refer to our article [Reducing CVEs in Elastic container images](https://www.elastic.co/blog/reducing-cves-in-elastic-container-images).

```bash subs=true
docker pull docker.elastic.co/beats/packetbeat-wolfi:{{stack-version}}
```


## Optional: Verify the image [_optional_verify_the_image]

You can use the [Cosign application](https://docs.sigstore.dev/cosign/installation/) to verify the Packetbeat Docker image signature.

% ::::{warning} subs=true
% Version {{stack-version}} of Packetbeat has not yet been released. No Docker image is currently available for Packetbeat {{stack-version}}.
% ::::


```sh subs=true
wget https://artifacts.elastic.co/cosign.pub
cosign verify --key cosign.pub docker.elastic.co/beats/packetbeat:{{stack-version}}
```

The `cosign` command prints the check results and the signature payload in JSON format:

```sh subs=true
Verification for docker.elastic.co/beats/packetbeat:{{stack-version}} --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The signatures were verified against the specified public key
```


## Run the Packetbeat setup [_run_the_packetbeat_setup]

::::{important}
A [known issue](https://github.com/elastic/beats/issues/42038) in version 8.17.0 prevents {{beats}} Docker images from starting when no options are provided. When running an image on that version, add an `--environment container` parameter to avoid the problem. This is planned to be addressed in issue [#42060](https://github.com/elastic/beats/pull/42060).
::::


Running Packetbeat with the setup command will create the index pattern and load visualizations , dashboards, and machine learning jobs.  Run this command:

```sh subs=true
docker run --rm \
--cap-add=NET_ADMIN \
docker.elastic.co/beats/packetbeat:{{stack-version}} \
setup -E setup.kibana.host=kibana:5601 \
-E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Kibana and Elasticsearch hosts and ports.
2. If you are using the hosted {{ess}} in {{ecloud}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using this syntax:


```shell
-E cloud.id=<Cloud ID from Elasticsearch Service> \
-E cloud.auth=elastic:<elastic password>
```


## Run Packetbeat on a read-only file system [_run_packetbeat_on_a_read_only_file_system]

If you’d like to run Packetbeat in a Docker container on a read-only file system, you can do so by specifying the `--read-only` option. Packetbeat requires a stateful directory to store application data, so with the `--read-only` option you also need to use the `--mount` option to specify a path to where that data can be stored.

For example:

```sh subs=true
docker run --rm \
  --mount type=bind,source=$(pwd)/data,destination=/usr/share/packetbeat/data \
  --read-only \
  docker.elastic.co/beats/packetbeat:{{stack-version}}
```


## Configure Packetbeat on Docker [_configure_packetbeat_on_docker]

The Docker image provides several methods for configuring Packetbeat. The conventional approach is to provide a configuration file via a volume mount, but it’s also possible to create a custom image with your configuration included.

### Example configuration file [_example_configuration_file]

Download this example configuration file as a starting point:

```sh
curl -L -O https://raw.githubusercontent.com/elastic/beats/master/deploy/docker/packetbeat.docker.yml
```


### Volume-mounted configuration [_volume_mounted_configuration]

One way to configure Packetbeat on Docker is to provide `packetbeat.docker.yml` via a volume mount. With `docker run`, the volume mount can be specified like this.

```sh subs=true
docker run -d \
  --name=packetbeat \
  --user=packetbeat \
  --volume="$(pwd)/packetbeat.docker.yml:/usr/share/packetbeat/packetbeat.yml:ro" \
  --cap-add="NET_RAW" \
  --cap-add="NET_ADMIN" \
  --network=host \
  docker.elastic.co/beats/packetbeat:{{stack-version}} \
  --strict.perms=false -e \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Elasticsearch hosts and ports.
2. If you are using the hosted {{ess}} in {{ecloud}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using the syntax shown earlier.



### Customize your configuration [_customize_your_configuration]

The `packetbeat.docker.yml` downloaded earlier should be customized for your environment. See [Configure](/reference/packetbeat/configuring-howto-packetbeat.md) for more details. Edit the configuration file and customize it to match your environment then re-deploy your Packetbeat container.


### Custom image configuration [_custom_image_configuration]

It’s possible to embed your Packetbeat configuration in a custom image. Here is an example Dockerfile to achieve this:

```dockerfile subs=true
FROM docker.elastic.co/beats/packetbeat:{{stack-version}}
COPY --chown=root:packetbeat packetbeat.yml /usr/share/packetbeat/packetbeat.yml
```


### Required network capabilities [_required_network_capabilities]

Under Docker, Packetbeat runs as a non-root user, but requires some privileged network capabilities to operate correctly. Ensure that the `NET_ADMIN` capability is available to the container.

```sh subs=true
docker run --cap-add=NET_ADMIN docker.elastic.co/beats/packetbeat:{{stack-version}}
```


### Capture traffic from the host system [_capture_traffic_from_the_host_system]

By default, Docker networking will connect the Packetbeat container to an isolated virtual network, with a limited view of network traffic. You may wish to connect the container directly to the host network in order to see traffic destined for, and originating from, the host system. With `docker run`, this can be achieved by specifying `--network=host`.

```sh subs=true
docker run --cap-add=NET_ADMIN --network=host docker.elastic.co/beats/packetbeat:{{stack-version}}
```

::::{note}
On Windows and MacOS, specifying `--network=host` will bind the container’s network interface to the virtual interface of Docker’s embedded Linux virtual machine, not to the physical interface of the host system.
::::




