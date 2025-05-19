---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/running-on-docker.html
---

# Run Filebeat on Docker [running-on-docker]

Docker images for Filebeat are available from the Elastic Docker registry. The base image is [centos:7](https://hub.docker.com/_/centos/).

A list of all published Docker images and tags is available at [www.docker.elastic.co](https://www.docker.elastic.co).

These images are free to use under the Elastic license. They contain open source and free commercial features and access to paid commercial features. [Start a 30-day trial](docs-content://deploy-manage/license/manage-your-license-in-self-managed-cluster.md) to try out all of the paid commercial features. See the [Subscriptions](https://www.elastic.co/subscriptions) page for information about Elastic license levels.

## Pull the image [_pull_the_image]

Obtaining Filebeat for Docker is as simple as issuing a `docker pull` command against the Elastic Docker registry.

% ::::{warning}
% Version {{stack-version}} of Filebeat has not yet been released. No Docker image is currently available for Filebeat {{stack-version}}.
% ::::


```sh subs=true
docker pull docker.elastic.co/beats/filebeat:{{stack-version}}
```

Alternatively, you can download other Docker images that contain only features available under the Apache 2.0 license. To download the images, go to [www.docker.elastic.co](https://www.docker.elastic.co).

As another option, you can use the hardened [Wolfi](https://wolfi.dev/) image. Using Wolfi images requires Docker version 20.10.10 or higher. For details about why the Wolfi images have been introduced, refer to our article [Reducing CVEs in Elastic container images](https://www.elastic.co/blog/reducing-cves-in-elastic-container-images).

```bash subs=true
docker pull docker.elastic.co/beats/filebeat-wolfi:{{stack-version}}
```


## Optional: Verify the image [_optional_verify_the_image]

You can use the [Cosign application](https://docs.sigstore.dev/cosign/installation/) to verify the Filebeat Docker image signature.

% ::::{warning}
% Version {{stack-version}} of Filebeat has not yet been released. No Docker image is currently available for Filebeat {{stack-version}}.
% ::::


```sh subs=true
wget https://artifacts.elastic.co/cosign.pub
cosign verify --key cosign.pub docker.elastic.co/beats/filebeat:{{stack-version}}
```

The `cosign` command prints the check results and the signature payload in JSON format:

```sh subs=true
Verification for docker.elastic.co/beats/filebeat:{{stack-version}} --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - Existence of the claims in the transparency log was verified offline
  - The signatures were verified against the specified public key
```


## Run the Filebeat setup [_run_the_filebeat_setup]

::::{important}
A [known issue](https://github.com/elastic/beats/issues/42038) in version 8.17.0 prevents {{beats}} Docker images from starting when no options are provided. When running an image on that version, add an `--environment container` parameter to avoid the problem. This is planned to be addressed in issue [#42060](https://github.com/elastic/beats/pull/42060).
::::


Running Filebeat with the setup command will create the index pattern and load visualizations , dashboards, and machine learning jobs.  Run this command:

```sh subs=true
docker run --rm \
docker.elastic.co/beats/filebeat:{{stack-version}} \
setup -E setup.kibana.host=kibana:5601 \
-E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Kibana and Elasticsearch hosts and ports.
2. If you are using the {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using this syntax:


```shell
-E cloud.id=<Cloud ID from Elastic Cloud Hosted> \
-E cloud.auth=elastic:<elastic password>
```


## Run Filebeat on a read-only file system [_run_filebeat_on_a_read_only_file_system]

If you’d like to run Filebeat in a Docker container on a read-only file system, you can do so by specifying the `--read-only` option. Filebeat requires a stateful directory to store application data, so with the `--read-only` option you also need to use the `--mount` option to specify a path to where that data can be stored.

For example:

```sh subs=true
docker run --rm \
  --mount type=bind,source=$(pwd)/data,destination=/usr/share/filebeat/data \
  --read-only \
  docker.elastic.co/beats/filebeat:{{stack-version}}
```


## Configure Filebeat on Docker [_configure_filebeat_on_docker]

The Docker image provides several methods for configuring Filebeat. The conventional approach is to provide a configuration file via a volume mount, but it’s also possible to create a custom image with your configuration included.

### Example configuration file [_example_configuration_file]

Download this example configuration file as a starting point:

```sh subs=true
curl -L -O https://raw.githubusercontent.com/elastic/beats/{{major-version}}/deploy/docker/filebeat.docker.yml
```


### Volume-mounted configuration [_volume_mounted_configuration]

One way to configure Filebeat on Docker is to provide `filebeat.docker.yml` via a volume mount. With `docker run`, the volume mount can be specified like this.

```sh subs=true
docker run -d \
  --name=filebeat \
  --user=root \
  --volume="$(pwd)/filebeat.docker.yml:/usr/share/filebeat/filebeat.yml:ro" \
  --volume="/var/lib/docker/containers:/var/lib/docker/containers:ro" \
  --volume="/var/run/docker.sock:/var/run/docker.sock:ro" \
  --volume="registry:/usr/share/filebeat/data:rw" \
  docker.elastic.co/beats/filebeat:{{stack-version}} filebeat -e --strict.perms=false \
  -E output.elasticsearch.hosts=["elasticsearch:9200"] <1> <2>
```

1. Substitute your Elasticsearch hosts and ports.
2. If you are using the {{ech}}, replace the `-E output.elasticsearch.hosts` line with the Cloud ID and elastic password using the syntax shown earlier.



### Customize your configuration [_customize_your_configuration]

The `filebeat.docker.yml` file you downloaded earlier is configured to deploy Beats modules based on the Docker labels applied to your containers.  See [Hints based autodiscover](/reference/filebeat/configuration-autodiscover-hints.md) for more details. Add labels to your application Docker containers, and they will be picked up by the Beats autodiscover feature when they are deployed.  Here is an example command for an Apache HTTP Server container with labels to configure the Filebeat and Metricbeat modules for the Apache HTTP Server:

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

It’s possible to embed your Filebeat configuration in a custom image. Here is an example Dockerfile to achieve this:

```dockerfile subs=true
FROM docker.elastic.co/beats/filebeat:{{stack-version}}
COPY --chown=root:filebeat filebeat.yml /usr/share/filebeat/filebeat.yml
```



