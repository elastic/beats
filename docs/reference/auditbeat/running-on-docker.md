---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/running-on-docker.html
---

# Run Auditbeat on Docker [running-on-docker]

Docker images for Auditbeat are available from the Elastic Docker registry. The base image is [centos:7](https://hub.docker.com/_/centos/).

A list of all published Docker images and tags is available at [www.docker.elastic.co](https://www.docker.elastic.co).

These images are free to use under the Elastic license. They contain open source and free commercial features and access to paid commercial features. [Start a 30-day trial](docs-content://deploy-manage/license/manage-your-license-in-self-managed-cluster.md) to try out all of the paid commercial features. See the [Subscriptions](https://www.elastic.co/subscriptions) page for information about Elastic license levels.

## Pull the image [_pull_the_image]

Obtaining Auditbeat for Docker is as simple as issuing a `docker pull` command against the Elastic Docker registry.

% ::::{warning} subs=true
% Version {{stack-version}} of Auditbeat has not yet been released. No Docker image is currently available for Auditbeat {{stack-version}}.
% ::::


```sh subs=true
docker pull docker.elastic.co/beats/auditbeat:{{stack-version}}
```

Alternatively, you can download other Docker images that contain only features available under the Apache 2.0 license. To download the images, go to [www.docker.elastic.co](https://www.docker.elastic.co).

As another option, you can use the hardened [Wolfi](https://wolfi.dev/) image. Using Wolfi images requires Docker version 20.10.10 or higher. For details about why the Wolfi images have been introduced, refer to our article [Reducing CVEs in Elastic container images](https://www.elastic.co/blog/reducing-cves-in-elastic-container-images).

```bash subs=true
docker pull docker.elastic.co/beats/auditbeat-wolfi:{{stack-version}}
```


## Optional: Verify the image [_optional_verify_the_image]

You can use the [Cosign application](https://docs.sigstore.dev/cosign/installation/) to verify the Auditbeat Docker image signature.

% ::::{warning} subs=true
% Version {{stack-version}} of Auditbeat has not yet been released. No Docker image is currently available for Auditbeat {{stack-version}}.
% ::::


```sh subs=true
wget https://artifacts.elastic.co/cosign.pub
cosign verify --key cosign.pub docker.elastic.co/beats/auditbeat:{{stack-version}}
```

The `cosign` command prints the check results and the signature payload in JSON format:

```sh subs=true
Verification for docker.elastic.co/beats/auditbeat:{{stack-version}} --
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
  docker.elastic.co/beats/auditbeat:{{stack-version}} \
  setup -E setup.kibana.host=kibana:5601 \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Kibana and Elasticsearch hosts and ports.
2. If you are using the {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using this syntax:


```shell
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
  docker.elastic.co/beats/auditbeat:{{stack-version}}
```


## Configure Auditbeat on Docker [_configure_auditbeat_on_docker]

The Docker image provides several methods for configuring Auditbeat. The conventional approach is to provide a configuration file via a volume mount, but it’s also possible to create a custom image with your configuration included.

### Example configuration file [_example_configuration_file]

Download this example configuration file as a starting point:

```sh subs=true
curl -L -O https://raw.githubusercontent.com/elastic/beats/{{major-version}}/deploy/docker/auditbeat.docker.yml
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
  docker.elastic.co/beats/auditbeat:{{stack-version}} -e \
  --strict.perms=false \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Elasticsearch hosts and ports.
2. If you are using the {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using the syntax shown earlier.



### Customize your configuration [_customize_your_configuration]

The `auditbeat.docker.yml` downloaded earlier should be customized for your environment. See [Configure](/reference/auditbeat/configuring-howto-auditbeat.md) for more details. Edit the configuration file and customize it to match your environment then re-deploy your Auditbeat container.


### Custom image configuration [_custom_image_configuration]

It’s possible to embed your Auditbeat configuration in a custom image. Here is an example Dockerfile to achieve this:

```dockerfile subs=true
FROM docker.elastic.co/beats/auditbeat:{{stack-version}}
COPY auditbeat.yml /usr/share/auditbeat/auditbeat.yml
```



## Special requirements [_special_requirements]

Under Docker, Auditbeat runs as a non-root user, but requires some privileged capabilities to operate correctly. Ensure that the `AUDIT_CONTROL` and `AUDIT_READ` capabilities are available to the container.

It is also essential to run Auditbeat in the host PID namespace.

```sh subs=true
docker run --cap-add=AUDIT_CONTROL --cap-add=AUDIT_READ --user=root --pid=host docker.elastic.co/beats/auditbeat:{{stack-version}}
```


