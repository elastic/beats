# Testing on OSX

A previous document regarding testing metricbeat at OSX existed, and have been moved to [./darwin.md](darwin.md)

# Testing on Linux

## Create Elasticsearch + Kibana instances

You can rely on your EK tuple of choice as long as it is addresable from the kubernetes cluster.

To boot a docker based EK this should suffice, be sure to replace image tags according to version:

```bash
# Run Elasticsearch
docker run --name es -d -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:7.3.0

# Run Kibana
docker run --name kibana -d --link es:elasticsearch -p 5601:5601 \
    docker.elastic.co/kibana/kibana:7.3.0
```

## Prepare assets

Generate binary and other assets for the beats branch you want to test, then copy them to a folder layed out to run:

- create folder `/home/myuser/playground/metricbeat`
- copy to that folder `metricbeat` binary and `metricbeat.yml`
- recursive copy `modules.d` from source repo to destination folder
- recursive copy `_meta/kibana.generated/{version}/dashboard/` to `kibana/{version}/dashboard/`

Configure `metricbeat.yml` and modules, do not use `localhost` to point to elasticsearch and kibana but the public ip of the host (one that will be routable from minikube)


## Create minikube cluster

Follow instructions https://kubernetes.io/docs/tasks/tools/install-minikube/ and start the minikube cluster.

Usually we should be ok with the kubernetes version that minikube creates, but you can force it by using `--kubernetes-version` flag.

```
minikube start --kubernetes-version v1.15.0
```

## Playground Pod

A playground Pod hosts the ubuntu container metricbeat will be running. A working playground is provided under [./01_playground](./01_playground) subfolder.

This file contains:

- a service account.
- a cluster role, if you are consuming kubernetes API resources, make sure that the APIGroup/Version, Resource and verb are listed here.
- a cluster role binding that links the service account to the service role
- an Ubuntu Pod:
  - uses `hostNetwork`, so it can reach ports at the host instance (for instance, the kubelet)
  - executes `sleep infinity`, so that it never exists, but does nothing
  - in order to be useful for filebeat, it mounts `/var/log/`, `/var/lib/docker/containers` and `/var/lib/filebeat-data`

At the time of writing this the Pod has been only used for 2 tests from the same person (hello), there is a lot of room for improvement.

To deploy the pod _as is_ you need to:

```
kubectl apply -f https://raw.githubusercontent.com/elastic/beats/master/metricbeat/module/kubernetes/_meta/test/docs/01_playground/playground-ubuntu.yaml
```

## Test


Binary and assets needed for the test that we prepared above need to be copied to the playground pod. Use `kubectl` to copy the directory, further iterations might only need to copy the changing assets.

Replace source folder and Pod namespace/name

```
kubectl cp --no-preserve  /home/myuser/playground/metricbeat playground:/metricbeat
```

Now you can exec into the container and launch metricbeat

```
 kubectl exec -ti playground /bin/bash

 cd /metricbeat
 ./metricbeat -c metricbeat.yml  -e

 ```

### Test Iterations

When copying new assets to an already used playground Pod, you will most probably run into an issue:
```
tar: metricbeat/kibana/7/dashboard/Metricbeat-aerospike-overview.json: Cannot open: Not a directory
tar: metricbeat/kibana/7/dashboard/Metricbeat-apache-overview.json: Cannot open: Not a directory
tar: metricbeat/kibana/7/dashboard/Metricbeat-ceph-overview.json: Cannot open: Not a directory
tar: metricbeat/kibana/7/dashboard/Metricbeat-consul-overview.json: Cannot open: Not a directory
```

I haven't looked much into this, there seems to be something going on when kubernetes untars the bundled directory. As a workaround, delete the metricbeat directory at the Pod before copying a new set of assets.

# Testing kubernetes loads

## Kube-state-metrics

Kube-state-metrics needs to be deployed for all the `state_` prefix metricsets at kubernetes. Yamls are to be found at the [upstream project](https://github.com/kubernetes/kube-state-metrics/tree/master/kubernetes)

Installing kube-state-metrics can be done either installing the yamls one by one from their remote location or cloning and installing the folder contents. Be sure to checkout the target release version before installing.

```
git clone git@github.com:kubernetes/kube-state-metrics.git
cd kube-state-metrics/

git checkout -b release-1.7 origin/release-1.7
kubectl apply -f kubernetes/
```


## Core components test

Testing core components (kubelet, apiserver, controller manager, scheduler) requires a diverse range of objects to be created. Using [Sonobuoy](https://github.com/heptio/sonobuoy) is the fastest path for testing,getting metrics and filling dashboards.

Refer to the documentation at Sonobuoy, at the time of this writing installing and running can be achieved with a couple commands

```
go get -u -v github.com/heptio/sonobuoy
sonobuoy run --wait
```

## Regular kubernetes components

You can find at [./02_objects](./02_objects) example kubernetes objects used during development and testing.
For now only a CronJob example is added, add your kubernetes object of choice under that folder if you consider it will be useful for other people when developing, testing and troubleshooting.

# Going further

- All improvements are welcome.
- Different ways to test are welcome and can live here side by side.
- Using kind seems to be a lot more lightweight.
- Probably some steps above can be tackled using [telepresence](https://www.telepresence.io/).
- Probably Sonobuoy can be replaced with kubernetes e2e tests.

