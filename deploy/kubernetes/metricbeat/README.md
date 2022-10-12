# Metricbeat

## Ship metrics from Kubernetes to Elasticsearch

### Settings


We use official [Metricbeat Docker images](https://www.docker.elastic.co/r/beats/metricbeat), as they allow external files' configuration. Our YAML manifests are the following:

* [metricbeat-daemonset-configmap.yaml](metricbeat-daemonset-configmap.yaml) to create the [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/) necessary to [metricbeat-daemonset.yaml](metricbeat-daemonset.yaml).

* [metricbeat-daemonset.yaml](metricbeat-daemonset.yaml) to create a [DeamonSet](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/). This way we ensure we get a running metricbeat daemon on each node of the cluster.
This file uses a set of environment variables to configure Elasticsearch output. We have two different sets:
  * [Self-managed](https://www.elastic.co/guide/en/beats/metricbeat/current/elasticsearch-output.html) Elasticsearch service:

    | Variable               | Default       | Description                          |
    |------------------------|---------------|--------------------------------------|
    | ELASTICSEARCH_HOST     | elasticsearch | Elasticsearch host                   |
    | ELASTICSEARCH_PORT     | 9200          | Elasticsearch port                   |
    | ELASTICSEARCH_USERNAME | elastic       | Elasticsearch username for HTTP auth |
    | ELASTICSEARCH_PASSWORD | changeme      | Elasticsearch password               |

  * Elasticsearch service on [Elastic Cloud](https://www.elastic.co/guide/en/beats/metricbeat/current/configure-cloud-id.html):

      | Variable           | Default | Description        |
      |--------------------|---------|--------------------|
      | ELASTIC_CLOUD_ID   |         | Elastic cloud ID   |
      | ELASTIC_CLOUD_AUTH |         | Elastic cloud auth |

* [metricbeat-role-binding.yaml](metricbeat-role-binding.yaml),[metricbeat-role.yaml](metricbeat-role.yaml) and [metricbeat-service-account.yaml](metricbeat-service-account.yaml) to define a user's permissions in our namespace (more information on this can be found [here](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)). Notice that the namespace we are using is `kube-system`, but this can be changed by updating the YAML manifests.



### Example

In this example, we will explore two different options to run our Kubernetes cluster:

- [For option 1](#Option-1.-Run-locally-on-Kind): Run locally using [Kind](https://kind.sigs.k8s.io/).
- [For option 2](#Option-2.-Run-on-GKE): Run on cloud using [GKE](https://cloud.google.com/kubernetes-engine).

In both options, our cluster will have more than one node.
This way we can see the usage of our Metricbeat DaemonSet.


#### Prerequisites

- [Kubectl](https://kubernetes.io/docs/tasks/tools/) to run commands against Kubernetes clusters.
- [For option 1](#Option-1.-Run-locally-on-Kind): [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/) for running local Kubernetes clusters.
- [For option 2](#Option-2.-Run-on-GKE): [gcloud Cli](https://cloud.google.com/sdk/docs/install).

#### Option 1. Run locally on Kind

The first thing we need to do is create a Kubernetes cluster. On kind, the default [configuration](https://kind.sigs.k8s.io/docs/user/configuration/) of a cluster generates only one node. As we are using a DaemonSet to run our Metricbeat, we will make a cluster run 4 nodes, so we can check our pods are running on all (or some) of them. To do this, we simply create the file `cluster-config.yaml`:
```YAML
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
- role: worker
```

Note that you can also specify your cluster's [nodes images](https://hub.docker.com/r/kindest/node/tags).

We now can create a cluster using the command below.

```
$ kind create cluster --config=cluster-config.yaml
```



Notice that our cluster was created using the configuration shown above. The name of our cluster is the default one, `kind`. To interact with the cluster, you only need to specify the cluster name as a context in kubectl:

```
$ kubectl cluster-info --context kind-kind
```


#### Option 2. Run on GKE

1. Create a GKE cluster. You can find information on how to do that [here](https://cloud.google.com/kubernetes-engine/docs/deploy-app-cluster).

2. Connect to the newly created cluster. You can do that by running the following:
```
$ gcloud container clusters get-credentials <cluster name> --zone <zone> --project <project name>
```


#### Run Metricbeat on Kubernetes

You can check the nodes of the newly created cluster by running the following:

```
$ kubectl get nodes
```

Metricbeat gets some metrics from [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics).
Clone the repository locally:

```
$ git clone git@github.com:kubernetes/kube-state-metrics.git
```

After that, you can deploy Kube State Metrics by running:

```
$ kubectl apply -f kube-state-metrics/examples/standard
```

To deploy Metricbeat to Kubernetes, clone this repository:

```
$ git clone git@github.com:elastic/beats.git
```

Don't forget to set the docker image version inside the [metricbeat-daemonset.yaml](metricbeat-daemonset.yaml).
Remember that you can find [Metricbeat Docker images here](https://www.docker.elastic.co/r/beats/metricbeat).
After that, run:

```
$ kubectl apply -f beats/deploy/kubernetes/metricbeat
```

You can check the pods of your cluster in the namespace `kube-system`. For that, you just run:

```
$ kubectl get pods -o wide --namespace=kube-system
```

You should be able to see each pod's node.
If you are using Linux/Unix/macOS you can request all pods whose names start with metricbeat:

```
$ kubectl get pods -o wide --namespace=kube-system | grep ^metricbeat
```


#### Visualizing data in Kibana

To visualize your data in Kibana, we will use Elastic Cloud. For that do the following:



1. [Log in](https://cloud.elastic.co/home) to your Elastic Cloud account.

2. Create a [deployment](https://www.elastic.co/guide/en/cloud/current/ec-create-deployment.html).
   While waiting, you are prompted to save the admin credentials for your deployment which provides you with superuser access to Elasticsearch.

3. On the deployment overview page, copy down the Cloud ID.

4. Set the environment variables on the [metricbeat-daemonset.yaml](metricbeat-daemonset.yaml) file:

    ```YAML
    - name: ELASTIC_CLOUD_ID
      value: <cloud_id>
    - name: ELASTIC_CLOUD_AUTH
      value: <username>:<password>
    ```

    If you already applied all the YAML manifest files you can just update this one:

    ```
    $ kubectl apply -f beats/deploy/kubernetes/metricbeat/metricbeat-daemonset.yaml
    ```

    Otherwise, make sure to apply all:

    ```
    $ kubectl apply -f beats/deploy/kubernetes/metricbeat
    ```

5. Navigate to the **Analytics** endpoint and select **Discover**. To see Metricbeat data, make sure the predefined `metricbeat-*` index pattern is selected.

6. To load a Kibana dashboard, you have to select **Dashboard** on the side menu.
For example, you can see data about your Metricbeat DaemonSet on the dashboard *[Metricbeat Kubernetes] DaemonSets*.
If by any chance, you don't have dashboards downloaded in Kibana, you can go [here](https://www.elastic.co/guide/en/beats/metricbeat/current/load-kibana-dashboards.html) to find a solution for that.


You should now be able to visualize your data on Kibana.

#### Visualizing your Kubernetes cluster

If you are curious about your cluster, you can use [this](https://k8slens.dev/) to visualize resources.
