# dockerlogbeat: a simple, a docker plugin for sending logs to elasticsearch


This code is a working alpha for a [docker logging plugin](https://docs.docker.com/engine/extend/plugins_logging/). With the proper config, it can send logs to elasticsearch.
## Build and install

To build and install, just run `mage create`


## Running

`docker run --log-driver=ossifrage/hellologdriver:0.0.1 --log-opt output.elasticsearch.hosts="172.18.0.2:9200" --log-opt output.elasticsearch.index="dockerbeat-test" -it debian:jessie /bin/bash`


## How it works

Logging plugins work by starting up an HTTP server that reads over a unix socket. When a container starts up that requests the logging plugin, a request is sent to `/LogDriver.StartLogging` with the name of the log handler and a struct containing the config of the container, including labels and other metadata. The actual log reading requires the file handle to be passed to a new routine which uses protocol buffers to read from the log handler. When the container stops, a request is sent to `/LogDriver.StopLogging`.



## Debugging on MacOS.

First, you need to shell into the VM that `runc` lives in. To do this, you need to find the tty for the VM. On newer versions of Docker For Mac, it's at: `~/Library/Containers/com.docker.docker/Data/vms/0/tty` On older versions, it's _somewhere_ else in `~/Library/Containers/com.docker.docker`. 


Once you find it, just run `screen ~/Library/Containers/com.docker.docker/Data/vms/0/tty`


The location of the logs AND the container base directory in the docker docs is wrong.


You can use this to find the list of plugins running on the host: `runc --root /containers/services/docker/rootfs/run/docker/plugins/runtime-root/plugins.moby/ list`

The logs are in `/var/log/docker`. If you want to make the logs useful, you need to find the ID of the plugin. Back on the darwin host, run `docker plugin inspect $name_of_plugin | grep Id` use the hash ID to grep through the logs: `grep 22bb02c1506677cd48cc1cfccc0847c1b602f48f735e51e4933001804f86e2e docker.*`


## Issues so far

- How do we want to integrate this with ingest pipelines?
- How do I pass a list of hosts via `--log-opts` ?
- improve error handling, try and pass more things to the HTTP handler if we can, so users can view them.
- Can we get this to send its own logs / health data to ES?
- Can the client logger know when its been closed?
- Pipeline close must be async.
- If something is down for too long, the publish operation can block,  preventing the FIFO queue from draining. We need more checks/parallelism to make it harder for the FIFO queue to get backed up.
- issues with `mage fmt`
- need to standardize make/mage targets and build tooling
- Sort out Vendor
- Settle licensing questions 