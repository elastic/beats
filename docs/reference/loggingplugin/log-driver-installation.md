---
navigation_title: "Install and configure"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/loggingplugin/current/log-driver-installation.html
---

# Install and configure the Elastic Logging Plugin [log-driver-installation]



## Before you begin [_before_you_begin]

Make sure your system meets the following prerequisites:

* Docker: Engine API 1.25 or later
* {{stack}}: Version 7.6.0 or later


## Step 1: Install the Elastic Logging Plugin plugin [_step_1_install_the_elastic_logging_plugin_plugin]

1. Install the plugin. You can install it from the Docker store (recommended), or build and install the plugin from source in the [beats](https://github.com/elastic/beats) GitHub repo.

    **To install from the Docker store:**

    ```sh subs=true
    docker plugin install elastic/elastic-logging-plugin:{{stack-version}}
    ```

    **To build and install from source:**

    [Set up your development environment](/extend/index.md#setting-up-dev-environment) and then run:

    ```shell
    cd x-pack/dockerlogbeat
    mage BuildAndInstall
    ```

2. If necessary, enable the plugin:

    ```sh subs=true
    docker plugin enable elastic/elastic-logging-plugin:{{stack-version}}
    ```

3. Verify that the plugin is installed and enabled:

    ```shell
    docker plugin ls
    ```

    The output should say something like:

    ```sh subs=true
    ID                  NAME                                   DESCRIPTION              ENABLED
    c2ff9d2cf090        elastic/elastic-logging-plugin:{{stack-version}}   A beat for docker logs   true
    ```



## Step 2: Configure the Elastic Logging Plugin [_step_2_configure_the_elastic_logging_plugin]

You can set configuration options for a single container, or for all containers running on the host. See [Configuration options](/reference/loggingplugin/log-driver-configuration.md) for a list of supported configuration options.

**To configure a single container:**

Pass configuration options at run time when you start the container. For example:

```sh subs=true
docker run --log-driver=elastic/elastic-logging-plugin:{{stack-version}} \
           --log-opt hosts="https://myhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           -it debian:jessie /bin/bash
```

**To configure all containers running on the host:**

Set configuration options in the Docker `daemon.json` configuration file. For example:

```json subs=true
{
  "log-driver" : "elastic/elastic-logging-plugin:{{stack-version}}",
  "log-opts" : {
    "hosts" : "https://myhost:9200",
    "user" : "myusername",
    "password" : "mypassword"
  }
}
```

::::{note}
The default location of the `daemon.json` file varies by platform. On Linux, the default location is `/etc/docker/daemon.json`. For more information, see the [Docker docs](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file).
::::


