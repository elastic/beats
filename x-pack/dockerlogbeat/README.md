# Elastic Logging Plugin: A Docker plugin for sending logs to Elasticsearch


A [Docker logging plugin](https://docs.docker.com/engine/extend/plugins_logging/) that ships container logs to Elasticsearch.

## Build and install

Build the plugin (requires Docker and [mage](https://github.com/magefile/mage#installation)):

```sh
mage build
```

Create and enable the plugin:

```sh
docker plugin rm dockerlogbeat-dev --force  # Remove old plugin
docker plugin create dockerlogbeat-dev build/package/elastic-logging-plugin
docker plugin enable dockerlogbeat-dev
```

## How to start the plugin

```sh
docker run --rm --log-driver=dockerlogbeat-dev --log-opt hosts="localhost:9200" alpine echo "hello"
```

## Config Options

The Plugin supports a number of Elasticsearch config options:

```sh
docker run --rm \
           --log-driver=dockerlogbeat-dev \
           --log-opt hosts="localhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           alpine echo "hello"
```

You can find complete documentation on the [Elastic site](https://www.elastic.co/guide/en/beats/loggingplugin/current/log-driver-configuration.html).

## Test with Elasticsearch

Start ES using [elastic-package](https://github.com/elastic/elastic-package) and load the env vars:

```sh
elastic-package stack up -d --services elasticsearch
eval "$(elastic-package stack shellinit)"
```

The plugin runs in an isolated rootfs that doesn't trust elastic-package's
self-signed CA by default. You can inject it into the plugin's CA bundle:

```sh
cat "$ELASTIC_PACKAGE_CA_CERT" >> build/package/elastic-logging-plugin/rootfs/etc/ssl/certs/ca-certificates.crt
docker plugin disable dockerlogbeat-dev --force
docker plugin rm dockerlogbeat-dev --force
docker plugin create dockerlogbeat-dev build/package/elastic-logging-plugin
docker plugin enable dockerlogbeat-dev
```

Run a container (the plugin uses host networking, so 127.0.0.1 reaches ES):

```sh
docker run --name dlb-es-test \
  --log-driver=dockerlogbeat-dev \
  --log-opt hosts="$ELASTIC_PACKAGE_ELASTICSEARCH_HOST" \
  --log-opt user="$ELASTIC_PACKAGE_ELASTICSEARCH_USERNAME" \
  --log-opt password="$ELASTIC_PACKAGE_ELASTICSEARCH_PASSWORD" \
  alpine sh -c 'for i in 1 2 3; do echo "test line $i"; done'

# Wait for async delivery, then verify:
sleep 5
curl -sk -u "$ELASTIC_PACKAGE_ELASTICSEARCH_USERNAME:$ELASTIC_PACKAGE_ELASTICSEARCH_PASSWORD" \
  "$ELASTIC_PACKAGE_ELASTICSEARCH_HOST/logs-docker-*/_search?pretty"
# Expected: 3 hits
```

Cleanup:

```sh
docker rm -f dlb-es-test
elastic-package stack down
```


## How it works

Logging plugins work by starting up an HTTP server that reads over a unix socket. When a container starts up that requests the logging plugin, a request is sent to `/LogDriver.StartLogging` with the name of the log handler and a struct containing the config of the container, including labels and other metadata. The actual log reading requires the file handle to be passed to a new routine which uses protocol buffers to read from the log handler. When the container stops, a request is sent to `/LogDriver.StopLogging`.



## Debugging on MacOS

First, you need to shell into the VM that `runc` lives in. To do this, you need to find the tty for the VM. On later versions of Docker For Mac, it's at: `~/Library/Containers/com.docker.docker/Data/vms/0/tty` On earlier versions, it's _somewhere_ else in `~/Library/Containers/com.docker.docker`.


Once you find it, run `screen ~/Library/Containers/com.docker.docker/Data/vms/0/tty`


The location of the logs AND the container base directory in the docker docs is wrong.


You can use this to find the list of plugins running on the host: `runc --root /containers/services/docker/rootfs/run/docker/plugins/runtime-root/plugins.moby/ list`

The logs are in `/var/log/docker`. If you want to make the logs useful, you need to find the ID of the plugin. Back on the darwin host, run `docker plugin inspect $name_of_plugin | grep Id` use the hash ID to grep through the logs: `grep 22bb02c1506677cd48cc1cfccc0847c1b602f48f735e51e4933001804f86e2e docker.*`


## Local logs

This plugin fully supports `docker logs`, and it maintains a local copy of logs that can be read without a connection to Elasticsearch. Unfortunately, due to the limitations in the docker plugin API, we can't "clean up" log files when a container is destroyed. The plugin mounts the `/var/lib/docker` directory on the host to write logs. This mount point can be changed using [Docker](https://docs.docker.com/engine/reference/commandline/plugin_set/#change-the-source-of-a-mount). The plugin can also be configured to do a "hard" cleanup and destroy logs when a container stops. To enable this, set the `DESTROY_LOGS_ON_STOP` environment var inside the plugin:
```
docker plugin set dockerlogbeat-dev DESTROY_LOGS_ON_STOP=true
```

You can also set `max-file`, `max-size` and `compress` using `--log-opts`
