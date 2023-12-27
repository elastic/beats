# README

This readme explain how to remote debug metricbeat running on docker/kubernetes from your laptop with your local IDE.

## Steps to run on docker:

0. Move into metricbeat folder from the root folder of this project. Note: For some reason even if we are building in a subfolder, the `substitutePath.to`  is still pointing to our root folder.

```bash
cd metricbeat
```

1. cross build image for linux

```bash
GOOS=linux GOARCH=amd64 go build -gcflags "-N -l" -o metricbeat main.go
```

2. buld docker container

```bash
docker build -t metricbeat-debugger-image -f Dockerfile.debug .
```

3. run docker container

```bash
docker run -p 56268:56268 --network elastic-package-stack_default -v $(pwd)/metric.docker.yml:/usr/share/metricbeat/metricbeat.yml metricbeat-debugger-image -c /usr/share/metricbeat/metricbeat.yml -e
```

You can customize the metricbeat configuration by mounting a different file instead of `$(pwd)/metric.docker.yml`.

4. Attach to the remote debugger via your local IDE. Follow [Attach to remote debugger](./README.md#attach-to-remote-debugger-via-your-local-ide)


## Steps to run on kubernetes:

Steps from 0 to 2 (included) are the same as `Steps to run on docker`

0. Move into metricbeat folder from the root folder of this project. Note: For some reason even if we are building in a subfolder, the `substitutePath.to`  is still pointing to our root folder.

```bash
cd metricbeat
```

1. cross build image for linux

```bash
GOOS=linux GOARCH=amd64 go build -gcflags "-N -l" -o metricbeat main.go
```

2. buld docker container

```bash
docker build -t metricbeat-debugger-image -f Dockerfile.debug .
```

3. load image into Kind in order to run on kubernetes

```bash
kind load docker-image metricbeat-debugger-image:latest
```

4. Edit `deploy/kubernetes/metricbeat-kubernetes.yaml` with these changes

```yaml
containers:
- name: metricbeat
  # image: docker.elastic.co/beats/metricbeat:8.2.0
  image: metricbeat-debugger-image:latest
  imagePullPolicy: Never
  args: [
    "-c", "/etc/metricbeat.yml",
    "-e",
    "-system.hostfs=/hostfs",
  ]
  ports:
    - containerPort: 56268
      hostPort: 56268
      protocol: TCP
```

Namely you need:
- change the docker image used
- add `imagePullPolicy` to pull the image from inside Kind
- add a `ports` to expose the port in order to remote debug from laptop

Compared to the docker example, here the metricbeat config is provided in the kubernetes manifest and mounted as a volume.

5. Apply the changes in kubernetes to run the metricbeat

```bash
kubectl apply -f metricbeat-kubernetes.yaml
```

6. Port forward from k8s to localhost

```bash
kubectl port-forward -n kube-system <pod-name> 56268:56268
```

where `<pod-name>` is the name of the pod running on k8s

7. Attach to the remote debugger via your local IDE. Follow [Attach to remote debugger](./README.md#attach-to-remote-debugger-via-your-local-ide)


## Attach to remote debugger via your local IDE

### Visual Studio Code
In order to attach to the remote debugger running on docker container or a k8s pod you need to provide a file `.vscode/launch.json` on your local machine with some configurations.

You can use the following template, but remember to replace `<absolute path>` with the absolute path of the root folder of your project.

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
					        "to": "<absolute path>"
                }
            ]
        }

    ]
}
```

## Goland/IntelliJ
More info at [here](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#attach-to-a-process-on-a-remote-machine)
