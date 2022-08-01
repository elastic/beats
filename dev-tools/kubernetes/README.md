# Readme

This folder container some dev tools that make it easier to develop and deploy filebeat and metricbeat running inside a Kubernetes cluster. This is especially useful when developing the metricbeat module `kubernetes` since it requires metricbeat to run inside a Kubernetes cluster in order to interact with kube-state-metrics and the Kubernetes APIs.

In details, a combination of Dockerfiles, Kubernetes manifests and Tiltfile make it possible to have features like:
- hot reloading of code running in Kubernetes, without re-applying the Kubernetes manifest
- remote debugging (with breakpoints) both metricbeat/filebeat running as a pod in a Kind Kubernetes cluster

[Tilt](https://tilt.dev/) is a dev toolkit for microservices.


## Setup
You can install Tilt by using the command

```shell
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
```


## How to run
The Tiltfile that orchestrates everything is located at `dev-tools/kubernetes/Tiltfile`. All the following commands need to be run from `dev-tools/kubernetes`, in the same folder where the Tiltfile is located.

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

If you want to use a remote debugger with Visual Studio code, you need to provide a `.vscode/launch.json` similar to the following file. In order for this to work on your laptop, you need to replace `<absolute_path_to_beats_folder>` with the absolute path of the root folder in this project. This file is currently not under git because it depends on the user configuration, it is only useful for VisualStudio Code and in a folder usually ignored by git.

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Connect to server",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "debugAdapter": "dlv-dap",
            "port": 56268,
            "host": "127.0.0.1",
            "showLog": true,
            "trace": "trace",
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
	                "from": "${workspaceFolder}",
                    "to": "<absolute_path_to_beats_folder>"
                }
            ]
        }

    ]
}
```


## Run vs debug mode
Currently the Tiltfile is configured in `run` mode

```python
beat(mode="run")
# beat(mode="debug")
```

This mode runs metricbeat like a single process in a pod.

If you want to switch to `debug` mode you have to modify the Tiltfile like the following code

```python
# beat(mode="run")
beat(mode="debug")
```

You can make this change while Tilt is running in the background. Tilt will
stop the running container and swap it with a debug container instead.

Both mode support `hot reloading`, meaning that if Tilt is running, when you make a change to
the source code, Tilt will:

1. Automatically compile the source code
2. Live sync the new binary to the running container
3. Restart the container to run the new binary
