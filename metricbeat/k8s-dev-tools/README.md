# Readme

K8s dev tools are a combination of Dockerfile, k8s manifest and Tiltfile.

[Tilt](https://tilt.dev/) is a dev toolkit for microservices.


## Setup
You can install Tilt by using the script

```shell
./metricbeat/k8s-dev-tools/get_tilt.sh
```


## How to run
The Tiltfile that orchestrates everything is located at `./metricbeat/Tiltfile` under the metricbeat folder of this repo. All the following commands need to be run from `./metricbeat`, in the same folder where the Tiltfile is located.

How to run Tilt

```shell
tilt up
```

This will open a terminal and optionally a web UI where you can interact with Tilt, see container logs and restart resources.

Once you are done with Tilt, you can simply `CTRL+C` from the open Tilt terminal. The resources that you started in k8s will still be running though.

If you want to remove all the k8s resources that you started with Tilt, you can run

```shell
tilt down
```


## Run vs debug mode
Currently the Tiltfile is configured in `run` mode

```python
metricbeat(mode="run")
# metricbeat(mode="debug")
```

This mode runs metricbeat like a single process in a pod.

If you want to switch to `debug` mode you have to modify the Tiltfile like the following code

```python
# metricbeat(mode="run")
metricbeat(mode="debug")
```

You can make this change while Tilt is running in the background. Tilt will
stop the running container and swap it with a debug container instead.

Both mode support `hot reloading`, meaning that if Tilt is running, when you make a change to
the source code, Tilt will:

1. Automatically compile the source code
2. Live sync the new binary to the running container
3. Restart the container to run the new binary
