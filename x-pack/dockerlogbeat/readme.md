# Elastic Logging Plugin: a docker plugin for sending logs to elasticsearch


This code is a working MVP for a [docker logging plugin](https://docs.docker.com/engine/extend/plugins_logging/). With the proper config, it can send logs to elasticsearch.
## Build and install

To build and install, just run `mage Package`. The build process happens entire within docker. The only external dependencies are [mage](https://github.com/magefile/mage#installation) and golang.


## Running

`docker run --log-driver=elastic/elastic-logging-plugin:8.0.0 --log-opt hosts="172.18.0.2:9200" -it debian:jessie /bin/bash`


## Config Options

The Plugin supports a number of Elasticsearch config options:

```
docker run --log-driver=elastic/{log-driver-alias}:{version} \
           --log-opt endpoint="myhost:9200" \
           --log-opt user="myusername" \
           --log-opt password="mypassword" \
           -it debian:jessie /bin/bash
```

You can find complete documentation on the [Elastic site](https://www.elastic.co/guide/en/beats/loggingplugin/current/log-driver-configuration.html).



## How it works

Logging plugins work by starting up an HTTP server that reads over a unix socket. When a container starts up that requests the logging plugin, a request is sent to `/LogDriver.StartLogging` with the name of the log handler and a struct containing the config of the container, including labels and other metadata. The actual log reading requires the file handle to be passed to a new routine which uses protocol buffers to read from the log handler. When the container stops, a request is sent to `/LogDriver.StopLogging`.



## Debugging on MacOS.

First, you need to shell into the VM that `runc` lives in. To do this, you need to find the tty for the VM. On newer versions of Docker For Mac, it's at: `~/Library/Containers/com.docker.docker/Data/vms/0/tty` On older versions, it's _somewhere_ else in `~/Library/Containers/com.docker.docker`. 


Once you find it, just run `screen ~/Library/Containers/com.docker.docker/Data/vms/0/tty`


The location of the logs AND the container base directory in the docker docs is wrong.


You can use this to find the list of plugins running on the host: `runc --root /containers/services/docker/rootfs/run/docker/plugins/runtime-root/plugins.moby/ list`

The logs are in `/var/log/docker`. If you want to make the logs useful, you need to find the ID of the plugin. Back on the darwin host, run `docker plugin inspect $name_of_plugin | grep Id` use the hash ID to grep through the logs: `grep 22bb02c1506677cd48cc1cfccc0847c1b602f48f735e51e4933001804f86e2e docker.*`


## Local logs

This plugin fully supports `docker logs`, and it maintains a local copy of logs that can be read without a connection to Elasticsearch. Unfortunately, due to the limitations in the docker plugin API, we can't "clean up" log files when a container is destroyed. The plugin mounts the `/var/lib/docker` directory on the host to write logs. This mount point can be changed via [Docker](https://docs.docker.com/engine/reference/commandline/plugin_set/#change-the-source-of-a-mount). The plugin can also be configured to do a "hard" cleanup and destroy logs when a container stops. To enable this, set the `DESTROY_LOGS_ON_STOP` environment var inside the plugin:
```
docker plugin set d805664c550e DESTROY_LOGS_ON_STOP=true
```

You can also set `max-file`, `max-size` and `compress` via `--log-opts`
