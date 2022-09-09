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

For more information on how to configure Tilt to run different scenarios look at the comments in the Tiltfile.

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
The behavior of the Tiltfile can be changed by calling the function `beat()` with different parameters:
- `beat`: `metricbeat` to test Metricbeat, `filebeat` to test Filebeat
- `mode`: `debug` to start a remote debugger that you can connect to from your IDE with hot reloading enabled, `run` to just run Metricbeat without a debugger but still with hot reloading enabled
- `arch`: `amd64` to build go binary for amd64 architecture, `arm64` to build go binary for arm64 (aka M1 Apple chip) architecture
- `k8s_env`: `kind` to run against a Kind cluster with no docker registry, `gcp` to use a docker registry on GCP. More info on docker registry on GCP at https://cloud.google.com/container-registry/docs/advanced-authentication#gcloud-helper.
- `k8s_cluster`: `single` to use a single node k8s cluster, `multi` to use a k8s with more than 1 node.
      if running on a multi-node cluster we expect to have at least 2 workers and a control plane node.
      A Beat in debugger mode will run on a node with a Tain `debugger=yes:NoSchedule`, while 1 Beat per node will run on all the other worker nodes.
      More info on Taints and Tolerations at https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/.
      You can add a taint with the following command:
          `kubectl taint nodes <node_name> debugger=yes:NoSchedule`

You can modify the Tiltfile while `tilt up` is running in the background. Tilt will try its best to update everything in place but depending what changes you made, you might want to `tilt down` and `tilt up` again just to make sure that everything was updated correctly.
