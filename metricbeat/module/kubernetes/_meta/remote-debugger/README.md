# README

This readme explain how to remote debug metricbeat running on docker/kubernetes from your laptop with VisualStudioCode. Other IDE can be used, here we only provide instructions for VisulStudioCode.

A common requirement for both remote debugging in docker or kubernetes is to have a file `.vscode/launch.json` on your local machine. 

Important Notice: Replace `<absolute path>` in the following config with the absolute path of the root folder.

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

3. run container

```bash
docker run -p 56268:56268 --network elastic-package-stack_default -v $(pwd)/metric.docker.yml:/usr/share/metricbeat/metricbeat.yml metricbeat-debugger-image -c /usr/share/metricbeat/metricbeat.yml -e
```

4. Run debugger from VisualStudio Code via `.vscode/launch.json`. Remember to add first some breakpoints


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
kind load docker-image --name kind-v1.23.5 metricbeat-debugger-image:latest
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

5. Port forward from k8s to localhost

```bash
kubectl port-forward <pod-name> 56268:56268
```

where `<pod-name>` is the name of the pod running on k8s

6. Run debugger from VisualStudio Code via `.vscode/launch.json`. Remember to add first some breakpoints. For example you can put a breakpoint at `metricbeat/cmd/root.go` at line 74 to stop at the very beginning of the metricbeat command
