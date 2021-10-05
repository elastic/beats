# POC Documentation

I generated this repo using the [beats development guide](https://www.elastic.co/guide/en/beats/devguide/current/newbeat-generate.html).
The kube-api call is based on the [k8s go-client example](https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration).

The interesting files are:
* `beater/kubebeat.go` - the beats logic
* `kubebeat.yml` - the beats config
* `Dockerfile` - runs the beat dockerized with debug flags
* `pod.yaml` - deploy the beat


## Running this example

This example assumes you have:
1. Elasticsearch with the default username & password (`elastic` & `changeme`) running on the default port (`http://localhost:9200`)
2. Kibana with running on the default port (`http://localhost:5601`)
3. Minikube cluster running locally (`minikube start`)

First compile the application for Linux:

    GOOS=linux go build

Then use the patch file to change the configuration for Minikube (or change the configuration according to your setup):

    patch kubebeat.yml kubebeat_minikube.yml.patch

Then package it to a docker image using the provided Dockerfile to run it on Kubernetes:

Running a [Minikube](https://minikube.sigs.k8s.io/docs/) cluster, you can build this image directly on the Docker engine of the Minikube node without pushing it to a registry. To build the image on Minikube:

    eval $(minikube docker-env)
    docker build -t kubebeat .

If you are not using Minikube, you should build this image and push it to a registry that your Kubernetes cluster can pull from.

If you have RBAC enabled on your cluster, use the following snippet to create role binding which will grant the default service account view permissions:

```
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default
```

Then, run the image in a Pod with a single instance Deployment:

    kubectl apply -f pod.yml

The example now sends requests to the Kubernetes API and sends to elastic events with pod information from the cluster every 5 seconds.

To validate check the logs:

    kubectl logs -f kubebeat-demo

Now go and check out the data on your Kibana! Make sure to add an index pattern `kubebeat*`

note: when changing the fields kibana will reject events dent from the kubebeat for not matching the existing scheme. make sure to delete the index when changing the event fields in your code.

### Clean up

To stop this example and clean up the pod, run:

    kubectl delete pod kubebeat-demo

### Open questions

1. Could we use some code from `kube-mgmt`/`gatekeeper`/`metricbeat` to do the kube-api querying and data management?
2. How should we integrate this to the agent?
3. ... many more

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
