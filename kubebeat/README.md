# POC Documentation

I generated this repo using the [beats development guide](https://www.elastic.co/guide/en/beats/devguide/current/newbeat-generate.html).
The kube-api call is based on the [k8s go-client example](https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration).

The interesting files are:
* `beater/kubebeat.go` - the beats logic
* `kubebeat.yml` - the beats config
* `Dockerfile` - runs the beat dockerized with debug flags
* `JUSTFILE` - just commander file


## Table of contents
- [POC Documentation](#poc-documentation)
  - [Table of contents](#table-of-contents)
  - [Prerequisites](#prerequisites)
  - [Running Kubebeat](#running-kubebeat-without-the-agent)
    - [Clean up](#clean-up)
    - [Remote Debugging](#remote-debugging)
- [{Beat}](#beat)
  - [Getting Started with {Beat}](#getting-started-with-beat)
    - [Requirements](#requirements)
    - [Init Project](#init-project)
    - [Build](#build)
    - [Run](#run)
    - [Test](#test)
    - [Update](#update)
    - [Cleanup](#cleanup)
    - [Clone](#clone)
  - [Packaging](#packaging)
  - [Build Elastic-Agent Docker with pre-packaged kubebeat](#build-elastic-agent-docker-with-pre-packaged-kubebeat)


## Prerequisites
**Please make sure that you run the following instructions within the `kubebeat` directory.**
1. [Just command runner](https://github.com/casey/just)
2. Elasticsearch with the default username & password (`elastic` & `changeme`) running on the default port (`http://localhost:9200`)
3. Kibana with running on the default port (`http://localhost:5601`)
4. Setup the local env:

```zsh
cd kubebeat & just setup-env
```

5. Clone the git submodule of the CIS rules:

```zsh
git submodule update --init
```

## Running Kubebeat (without the agent)

Build & deploy kubebeat:

```zsh
just build-deploy-kubebeat
```

To validate check the logs:

```zsh
kubectl logs -f --selector="k8s-app=kubebeat"  -n kube-system
```

Now go and check out the data on your Kibana! Make sure to add a kibana dataview `logs-k8s_cis.result-*`

note: when changing the fields kibana will reject events dent from the kubebeat for not matching the existing scheme. make sure to delete the index when changing the event fields in your code.

### Clean up

To stop this example and clean up the pod, run:
```zsh
kubectl delete -f deploy/k8s/kubebeat-ds-local.yaml -n kube-system
```
### Remote Debugging

Build & Deploy remote debug docker:

```zsh
just build-deploy-kubebeat-debug
```

After running the pod, expose the relevant ports:
```zsh
kubectl port-forward ${pod-name} -n kube-system 40000:40000 8080:8080
```

The app will wait for the debugger to connect before starting

```zsh
kubectl logs -f --selector="k8s-app=kubebeat" -n kube-system
```

Use your favorite IDE to connect to the debugger on `localhost:40000` (for example [Goland](https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html#step-3-create-the-remote-run-debug-configuration-on-the-client-computer))

# {Beat}

Welcome to {Beat}.

Ensure that this folder is at the following location:
`${GOPATH}/src/github.com/elastic/beats/v7/kubebeat`

## Getting Started with {Beat}

### Requirements

* [Golang](https://golang.org/dl/) 1.7

### Init Project
To get running with {Beat} and also install the
dependencies, run the following command:

```
make update
```

It will create a clean git history for each major step. Note that you can always rewrite the history if you wish before pushing your changes.

To push {Beat} in the git repository, run the following commands:

```
git remote set-url origin https://github.com/elastic/beats/v7/kubebeat
git push origin master
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

### Build

To build the binary for {Beat} run the command below. This will generate a binary
in the same directory with the name kubebeat.

```
make
```


### Run

To run {Beat} with debugging output enabled, run:

```
./kubebeat -c kubebeat.yml -e -d "*"
```

### Test

To test {Beat}, run the following command:

```
make testsuite
```

alternatively:
```
make unit-tests
make system-tests
make integration-tests
make coverage-report
```

The test coverage is reported in the folder `./build/coverage/`

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `fields.yml` by running the following command.

```
make update
```


### Cleanup

To clean  {Beat} source code, run the following command:

```
make fmt
```

To clean up the build directory and generated artifacts, run:

```
make clean
```


### Clone

To clone {Beat} from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/src/github.com/elastic/beats/v7/kubebeat
git clone https://github.com/elastic/beats/v7/kubebeat ${GOPATH}/src/github.com/elastic/beats/v7/kubebeat
```


For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Packaging

The beat frameworks provides tools to crosscompile and package your beat for different platforms. This requires [docker](https://www.docker.com/) and vendoring as described above. To build packages of your beat, run the following command:

```
make release
```

This will fetch and create all images required for the build process. The whole process to finish can take several minutes.

## Build Elastic-Agent Docker with pre-packaged kubebeat


**1.Build Elastic-Agent Docker**

1. initialise git submodule for rego rules:
```
$ git submodule update --init
```
2. Access the Elastic-Agent dir
```
$ cd x-pack/elastic-agent
```
3. Build & deploy the elastic-agent docker( You might need to increase docker engine resources on your docker-engine)
```
$ just build-deploy-agent # It takes a while on the first execution.
```
4. Once command is finished, Verify the agent is running on your machine

```zsh
$ kubectl get po --selector="app=elastic-agent" --all-namespaces -o wide

NAMESPACE     NAME                  READY   STATUS    RESTARTS   AGE    IP           NODE                      NOMINATED NODE   READINESS GATES
kube-system   elastic-agent-9fpdz   1/1     Running   0          5h8m   172.20.0.2   kind-mono-control-plane   <none>           <none>


```

Note: Check the jusfile for all available commands for build or deploy `$ just --summary`
</br>

