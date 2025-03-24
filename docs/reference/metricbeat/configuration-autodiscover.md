---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-autodiscover.html
---

# Autodiscover [configuration-autodiscover]

When you run applications on containers, they become moving targets to the monitoring system. Autodiscover allows you to track them and adapt settings as changes happen. By defining configuration templates, the autodiscover subsystem can monitor services as they start running.

You define autodiscover settings in the  `metricbeat.autodiscover` section of the `metricbeat.yml` config file. To enable autodiscover, you specify a list of providers.


## Providers [_providers]

Autodiscover providers work by watching for events on the system and translating those events into internal autodiscover events with a common format. When you configure the provider, you can optionally use fields from the autodiscover event to set conditions that, when met, launch specific configurations.

On start, Metricbeat will scan existing containers and launch the proper configs for them. Then it will watch for new start/stop events. This ensures you don’t need to worry about state, but only define your desired configs.


#### Docker [_docker_2]

The Docker autodiscover provider watches for Docker containers to start and stop.

It has the following settings:

`host`
:   (Optional) Docker socket (UNIX or TCP socket). It uses `unix:///var/run/docker.sock` by default.

`ssl`
:   (Optional) SSL configuration to use when connecting to the Docker socket.

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container, disabled by default.

`labels.dedot`
:   (Optional) Default to be false. If set to true, replace dots in labels with `_`.

These are the fields available within config templating. The `docker.*` fields will be available on each emitted event. event:

* host
* port
* docker.container.id
* docker.container.image
* docker.container.name
* docker.container.labels

For example:

```yaml
{
  "host": "10.4.15.9",
  "port": 6379,
  "docker": {
    "container": {
      "id": "382184ecdb385cfd5d1f1a65f78911054c8511ae009635300ac28b4fc357ce51"
      "name": "redis",
      "image": "redis:3.2.11",
      "labels": {
        "io.kubernetes.pod.namespace": "default"
        ...
      }
    }
  }
}
```

You can define a set of configuration templates to be applied when the condition matches an event. Templates define a condition to match on autodiscover events, together with the list of configurations to launch when this condition happens.

Conditions match events from the provider. Providers use the same format for [Conditions](/reference/metricbeat/defining-processors.md#conditions) that processors use.

Configuration templates can contain variables from the autodiscover event. They can be accessed under the `data` namespace. For example, with the example event, "`${data.port}`" resolves to `6379`.

Metricbeat supports templates for modules:

```yaml
metricbeat.autodiscover:
  providers:
    - type: docker
      labels.dedot: true
      templates:
        - condition:
            contains:
              docker.container.image: redis
          config:
            - module: redis
              metricsets: ["info", "keyspace"]
              hosts: "${data.host}:6379"
```

This configuration launches a `redis` module for all containers running an image with `redis` in the name. `labels.dedot` defaults to be `true` for docker autodiscover, which means dots in docker labels are replaced with *_* by default.

Also Metricbeat autodiscover supports leveraging [Secrets keystore](/reference/metricbeat/keystore.md) in order to retrieve sensitive data like passwords. Here is an example of how a configuration using keystore would look like:

```yaml
metricbeat.autodiscover:
  providers:
    - type: docker
      labels.dedot: true
      templates:
        - condition:
            contains:
              docker.container.image: redis
          config:
            - module: redis
              metricsets: ["info", "keyspace"]
              hosts: "${data.host}:6379"
              password: "${REDIS_PASSWORD}"
```

where `REDIS_PASSWORD` is a key stored in local keystore of Metricbeat.


#### Kubernetes [_kubernetes]

The Kubernetes autodiscover provider watches for Kubernetes nodes, pods, services to start, update, and stop.

The `kubernetes` autodiscover provider has the following configuration settings:

`node`
:   (Optional) Specify the node to scope metricbeat to in case it cannot be accurately detected, as when running metricbeat in host network mode.

`namespace`
:   (Optional) Select the namespace from which to collect the events from the resources. If it is not set, the provider collects them from all namespaces. It is unset by default. The namespace configuration only applies to kubernetes resources that are namespace scoped and if `unique` field is set to `false`.

`cleanup_timeout`
:   (Optional) Specify the time of inactivity before stopping the running configuration for a container, disabled by default.

`kube_config`
:   (Optional) Use given config file as configuration for Kubernetes client. If kube_config is not set, KUBECONFIG environment variable will be checked and if not present it will fall back to InCluster.

`kube_client_options`
:   (Optional) Additional options can be configured for Kubernetes client. Currently client QPS and burst are supported, if not set Kubernetes client’s [default QPS and burst](https://pkg.go.dev/k8s.io/client-go/rest#pkg-constants) will be used. Example:

```yaml
      kube_client_options:
        qps: 5
        burst: 10
```

`resource`
:   (Optional) Select the resource to do discovery on. Currently supported Kubernetes resources are `pod`, `service` and `node`. If not configured `resource` defaults to `pod`.

`scope`
:   (Optional) Specify at what level autodiscover needs to be done at. `scope` can either take `node` or `cluster` as values. `node` scope allows discovery of resources in the specified node. `cluster` scope allows cluster wide discovery. Only `pod` and `node` resources can be discovered at node scope.

`add_resource_metadata`
:   (Optional) Specify filters and configration for the extra metadata, that will be added to the event. Configuration parameters:

    * `node` or `namespace`: Specify labels and annotations filters for the extra metadata coming from node and namespace. By default all labels are included while annotations are not. To change default behaviour `include_labels`, `exclude_labels` and `include_annotations` can be defined. Those settings are useful when storing labels and annotations that require special handling to avoid overloading the storage output. Note: wildcards are not supported for those settings. The enrichment of `node` or `namespace` metadata can be individually disabled by setting `enabled: false`.
    * `deployment`: If resource is `pod` and it is created from a `deployment`, by default the deployment name isn’t added, this can be enabled by setting `deployment: true`.
    * `cronjob`: If resource is `pod` and it is created from a `cronjob`, by default the cronjob name isn’t added, this can be enabled by setting `cronjob: true`.

        Example:


```yaml
      add_resource_metadata:
        namespace:
          include_labels: ["namespacelabel1"]
        node:
          include_labels: ["nodelabel2"]
          include_annotations: ["nodeannotation1"]
        # deployment: false
        # cronjob: false
```

`unique`
:   (Optional) Defaults to `false`. Marking an autodiscover provider as unique results into making the provider to enable the provided templates only when it will gain the leader lease. This setting can only be combined with `cluster` scope. When `unique` is enabled, `resource` and `add_resource_metadata` settings are not taken into account.

`leader_lease`
:   (Optional) Defaults to `metricbeat-cluster-leader`. This will be name of the lock lease. One can monitor the status of the lease with `kubectl describe lease beats-cluster-leader`. Different Beats that refer to the same leader lease will be competitors in holding the lease and only one will be elected as leader each time.

`leader_leaseduration`
:   (Optional) Duration that non-leader candidates will wait to force acquire the lease leadership. Defaults to `15s`.

`leader_renewdeadline`
:   (Optional) Duration that the leader will retry refreshing its leadership before giving up. Defaults to `10s`.

`leader_retryperiod`
:   (Optional) Duration that the metricbeat instances running to acquire the lease should wait between tries of actions. Defaults to `2s`.

Configuration templates can contain variables from the autodiscover event. These variables can be accessed under the `data` namespace, e.g. to access Pod IP: `${data.kubernetes.pod.ip}`.

These are the fields available within config templating. The `kubernetes.*` fields will be available on each emitted event:


##### Generic fields: [_generic_fields]

* host


##### Pod specific: [_pod_specific]

| Key | Type | Description |
| --- | --- | --- |
| `port` | `string` | Pod port. If pod has multiple ports exposed should be used `ports.<port-name>` instead |
| `kubernetes.namespace` | `string` | Namespace, where the Pod is running |
| `kubernetes.namespace_uuid` | `string` | UUID of the Namespace, where the Pod is running |
| `kubernetes.namespace_annotations.*` | `object` | Annotations of the Namespace, where the Pod is running. Annotations should be used in not dedoted format, e.g. `kubernetes.namespace_annotations.app.kubernetes.io/name` |
| `kubernetes.pod.name` | `string` | Name of the Pod |
| `kubernetes.pod.uid` | `string` | UID of the Pod |
| `kubernetes.pod.ip` | `string` | IP of the Pod |
| `kubernetes.labels.*` | `object` | Object of the Pod labels. Labels should be used in not dedoted format, e.g. `kubernetes.labels.app.kubernetes.io/name` |
| `kubernetes.annotations.*` | `object` | Object of the Pod annotations. Annotations should be used in not dedoted format, e.g. `kubernetes.annotations.test.io/test` |
| `kubernetes.container.name` | `string` | Name of the container |
| `kubernetes.container.runtime` | `string` | Runtime of the container |
| `kubernetes.container.id` | `string` | ID of the container |
| `kubernetes.container.image` | `string` | Image of the container |
| `kubernetes.node.name` | `string` | Name of the Node |
| `kubernetes.node.uid` | `string` | UID of the Node |
| `kubernetes.node.hostname` | `string` | Hostname of the Node |


##### Node specific: [_node_specific]

| Key | Type | Description |
| --- | --- | --- |
| `kubernetes.labels.*` | `object` | Object of labels of the Node |
| `kubernetes.annotations.*` | `object` | Object of annotations of the Node |
| `kubernetes.node.name` | `string` | Name of the Node |
| `kubernetes.node.uid` | `string` | UID of the Node |
| `kubernetes.node.hostname` | `string` | Hostname of the Node |


##### Service specific: [_service_specific]

| Key | Type | Description |
| --- | --- | --- |
| `port` | `string` | Service port |
| `kubernetes.namespace` | `string` | Namespace of the Service |
| `kubernetes.namespace_uuid` | `string` | UUID of the Namespace of the Service |
| `kubernetes.namespace_annotations.*` | `object` | Annotations of the Namespace of the Service. Annotations should be used in not dedoted format, e.g. `kubernetes.namespace_annotations.app.kubernetes.io/name` |
| `kubernetes.labels.*` | `object` | Object of the Service labels |
| `kubernetes.annotations.*` | `object` | Object of the Service annotations |
| `kubernetes.service.name` | `string` | Name of the Service |
| `kubernetes.service.uid` | `string` | UID of the Service |

If the `include_annotations` config is added to the provider config, then the list of annotations present in the config are added to the event.

If the `include_labels` config is added to the provider config, then the list of labels present in the config will be added to the event.

If the `exclude_labels` config is added to the provider config, then the list of labels present in the config will be excluded from the event.

if the `labels.dedot` config is set to be `true` in the provider config, then `.` in labels will be replaced with `_`. By default it is `true`.

if the `annotations.dedot` config is set to be `true` in the provider config, then `.` in annotations will be replaced with `_`. By default it is `true`.

::::{note}
Starting from 8.6 release `kubernetes.labels.*` used in config templating are not dedoted regardless of `labels.dedot` value. This config parameter only affects the fields added in the final Elasticsearch document. For example, for a pod with label `app.kubernetes.io/name=ingress-nginx` the matching condition should be `condition.equals: kubernetes.labels.app.kubernetes.io/name: "ingress-nginx"`. If `labels.dedot` is set to `true`(default value) the label will be stored in Elasticsearch as `kubernetes.labels.app_kubernetes_io/name`. The same applies for kubernetes annotations.
::::


For example:

```yaml
{
  "host": "172.17.0.21",
  "port": 9090,
  "kubernetes": {
    "container": {
      "id": "bb3a50625c01b16a88aa224779c39262a9ad14264c3034669a50cd9a90af1527",
      "image": "prom/prometheus",
      "name": "prometheus"
    },
    "labels": {
      "project": "prometheus",
      ...
    },
    "namespace": "default",
    "node": {
      "name": "minikube"
    },
    "pod": {
      "name": "prometheus-2657348378-k1pnh"
    }
  },
}
```

Example:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      scope: cluster
      node: ${NODE_NAME}
      unique: true
      identifier: leader-election-metricbeat
      templates:
        - config:
            - module: kubernetes
              hosts: ["kube-state-metrics:8080"]
              period: 10s
              add_metadata: true
              metricsets:
                - state_node
```

The above configuration when deployed on one or more Metribceat instances will enable `state_node` metricset only for the Metricbeat instance that will gain the leader lease/lock. With this deployment strategy we can ensure that cluster-wide metricsets are only enabled by one Beat instance when deploying a Beat as DaemonSet.

Metricbeat supports templates for modules:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      include_annotations: ["prometheus.io.scrape"]
      templates:
        - condition:
            contains:
              kubernetes.annotations.prometheus.io/scrape: "true"
          config:
            - module: prometheus
              metricsets: ["collector"]
              hosts: "${data.host}:${data.port}"
```

This configuration launches a `prometheus` module for all containers of pods annotated `prometheus.io/scrape=true`.


#### Manually Defining Ports with Kubernetes [_manually_defining_ports_with_kubernetes]

Declare exposed ports in your pod spec if possible. Otherwise, you will need to use multiple templates with complex filtering rules. The `{port}` variable will not be present, and you will need to hardcode ports. Example: `{data.host}:1234`

When ports are not declared, Autodiscover generates a config using your provided template once per pod, and once per container. These generated configs are de-duplicated after they are generated. If the generated configs for multiple containers are identical, they will be merged into one config.

Pods share an identical host. If only the `{data.host}` variable is interpolated, then one config will be generated per host. The configs will be identical. After they are de-duplicated, only one will be used.

In order to target one specific exposed port `{data.host}:{data.ports.web}` can be used in template config, where `web` is the name of the exposed container port.


#### Metricbeat Autodiscover Secret Management [kubernetes-secrets]


##### Local Keystore [_local_keystore]

Metricbeat autodiscover supports leveraging [Secrets keystore](/reference/metricbeat/keystore.md) in order to retrieve sensitive data like passwords. Here is an example of how a configuration using keystore would look like:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      templates:
        - condition:
            contains:
              kubernetes.labels.app: "redis"
          config:
            - module: redis
              metricsets: ["info", "keyspace"]
              hosts: "${data.host}:6379"
              password: "${REDIS_PASSWORD}"
```

where `REDIS_PASSWORD` is a key stored in local keystore of Metricbeat.


#### Kubernetes Secrets [_kubernetes_secrets]

Metricbeat autodiscover supports leveraging [Kubernetes secrets](https://kubernetes.io/docs/concepts/configuration/secret/) in order to retrieve sensitive data like passwords. In order to enable this feature add the following section in Metricbeat’s `ClusterRole` rules:

```yaml
- apiGroups: [""]
  resources:
    - secrets
  verbs: ["get"]
```

::::{warning}
The above rule will give permission to Metricbeat Pod to access Kubernetes Secrets API. This means that anyone who have access to Metricbeat Pod (`kubectl exec` for example) will be able to access Kubernetes Secrets API and get a specific secret no matter which namespace it belongs to. This option should be carefully considered, specially when used with hints.
::::


One option to give permissions only for one namespace, and not cluster-scoped, is to use a specific Role for a targeted namespace so as to better control access:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: marketing-team
  name: secret-reader
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["secrets"]
  verbs: ["get"]
```

One can find more info about Role and ClusterRole in the official Kubernetes [documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

Here is an example of how a configuration using Kubernetes secrets would look like:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      templates:
        - condition:
            contains:
              kubernetes.labels.app: "redis"
          config:
            - module: redis
              metricsets: ["info", "keyspace"]
              hosts: "${data.host}:6379"
              password: "${kubernetes.default.somesecret.value}"
```

where `kubernetes.default.somesecret.value` specifies a key stored as Kubernetes secret as following:

1. Kubernetes Namespace: `default`
2. Kubernetes Secret Name: `somesecret`
3. Secret Data Key: `value`

This secret can be created in a Kubernetes environment using the following command:

```yaml
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: somesecret
type: Opaque
data:
  value: $(echo -n "passpass" | base64)
EOF
```

Note that Pods can only consume secrets that belong to the same Kubernetes namespace. For instance if Pod `my-redis` is running under `staging` namespace, it cannot access a secret under `testing` namespace for example `kubernetes.testing.xxx.yyy`.


#### Jolokia [_jolokia]

The Jolokia autodiscover provider uses Jolokia Discovery to find agents running in your host or your network.

The configuration of this provider consists in a set of network interfaces, as well as a set of templates as in other providers. The network interfaces will be the ones used for discovery probes, each item of `interfaces` has these settings:

`name`
:   the name of the interface (e.g. `br0`), it can contain a wildcard as suffix to apply the same settings to multiple network interfaces of the same type (e.g. `br*`).

`interval`
:   time between probes (defaults to 10s)

`grace_period`
:   time since the last reply to consider an instance stopped (defaults to 30s)

`probe_timeout`
:   max time to wait for responses since a probe is sent (defaults to 1s)

Jolokia Discovery mechanism is supported by any Jolokia agent since version 1.2.0, it is enabled by default when Jolokia is included in the application as a JVM agent, but disabled in other cases as the OSGI or WAR (Java EE) agents. In any case, this feature is controlled with two properties:

* `discoveryEnabled`, to enable the feature
* `discoveryAgentUrl`, if set, this is the URL announced by the agent when being discovered, setting this parameter implicitly enables the feature

There are multiple ways of setting these properties, and they can vary from application to application, please refer to the documentation of your application to find the more suitable way to set them in your case.

Jolokia Discovery is based on UDP multicast requests. Agents join the multicast group 239.192.48.84, port 24884, and discovery is done by sending queries to this group. You have to take into account that UDP traffic between Metricbeat and the Jolokia agents has to be allowed. Also notice that this multicast address is in the 239.0.0.0/8 range, that is reserved for private use within an organization, so it can only be used in private networks.

These are the available fields during within config templating. The `jolokia.*` fields will be available on each emitted event.

* jolokia.agent.id
* jolokia.agent.version
* jolokia.secured
* jolokia.server.product
* jolokia.server.vendor
* jolokia.server.version
* jolokia.url

Metricbeat supports templates for modules:

```yaml
metricbeat.autodiscover:
  providers:
    - type: jolokia
      interfaces:
      - name: br*
        interval: 5s
        grace_period: 10s
      - name: en*
      templates:
      - condition:
          contains:
            jolokia.server.product: "tomcat"
        config:
        - module: jolokia
          metricsets: ["jmx"]
          hosts: "${data.jolokia.url}"
          namespace: test
          jmx.mappings:
          - mbean: "java.lang:type=Runtime"
            attributes:
            - attr: Uptime
              field: uptime
```

This configuration starts a jolokia module that collects the uptime of each `tomcat` instance discovered. Discovery probes are sent using all interfaces starting with `br` and `en`, for the `br` interfaces the `interval` and `grace_period` is reduced to 5 and 10 seconds respectively.


#### Amazon EC2s [_amazon_ec2s]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


The Amazon EC2 autodiscover provider discovers [EC2 instances](https://aws.amazon.com/ec2/). This is useful for users to launch Metricbeat modules to monitor services running on AWS EC2 instances.

For example, you can use this provider to gather MySQL metrics from MySQL servers running on EC2 instances that have a specific tag, `service: mysql`.

This provider will load AWS credentials using the standard AWS environment variables and shared credentials files see [Best Practices for Managing AWS Access Keys](https://docs.aws.amazon.com/general/latest/gr/aws-access-keys-best-practices.html) for more information. If you do not wish to use these, you may explicitly set the `access_key_id` and `secret_access_key` variables.

These are the available fields during within config templating. The `aws.ec2.*` fields and `cloud.*` fields will be available on each emitted event.

* cloud.availability_zone
* cloud.instance.id
* cloud.machine.type
* cloud.provider
* cloud.region
* aws.ec2.architecture
* aws.ec2.image.id
* aws.ec2.kernel.id
* aws.ec2.monitoring.state
* aws.ec2.private.dns_name
* aws.ec2.private.ip
* aws.ec2.public.dns_name
* aws.ec2.public.ip
* aws.ec2.root_device_name
* aws.ec2.state.code
* aws.ec2.state.name
* aws.ec2.subnet.id
* aws.ec2.tags
* aws.ec2.vpc.id

Metricbeat supports templates for modules:

```yaml
metricbeat.autodiscover:
  providers:
    - type: aws_ec2
      period: 1m
      credential_profile_name: elastic-beats
      templates:
        - condition:
            equals:
              aws.ec2.tags.service: "mysql"
          config:
            - module: mysql
              metricsets: ["status", "galera_status"]
              period: 10s
              hosts: ["root:password@tcp(${data.aws.ec2.public.ip}:3306)/"]
              username: root
              password: password
```

This autodiscover provider takes our standard AWS credentials options. With this configuration, `mysql` metricbeat module will be launched for all EC2 instances that have `service: mysql` as a tag.

This autodiscover provider takes our standard [AWS credentials options](/reference/metricbeat/metricbeat-module-aws.md#aws-credentials-config).
