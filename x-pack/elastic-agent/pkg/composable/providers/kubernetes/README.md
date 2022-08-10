##  Autodiscover provider in Elastic Agent

https://www.elastic.co/guide/en/fleet/8.0/dynamic-input-configuration.html#dynamic-providers

Currently Kubernetes dynamic provider can only be configured in [standalone](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/deploy/kubernetes/elastic-agent-standalone-kubernetes.yaml#L24) agent.
In fleet managed agent it is enabled by default with default values.

Template based autodiscover of Kubernetes resources is only supported in standalone mode as of now.
It is not part of the Kubernetes Integration yet.

Hints based autodiscover is not supported yet.

### Template based autodiscover

Example:
As an example we will use again redis module.
In agent.yml(configmap) an extra input block needs to be added.
```
# Add extra input blocks here, based on conditions
# so as to automatically identify targeted Pods and start monitoring them
# using a predefined integration. For instance:
- name: redis
  type: redis/metrics
  use_output: default
  meta:
    package:
      name: redis
      version: 0.3.6
  data_stream:
    namespace: default
  streams:
    - data_stream:
        dataset: redis.info
        type: metrics
      metricsets:
        - info
      hosts:
        - '${kubernetes.pod.ip}:6379'
      idle_timeout: 20s
      maxconn: 10
      network: tcp
      period: 10s
      condition: ${kubernetes.pod.labels.app} == 'redis'
```

What makes this input block dynamic are the variables hosts and condition.
`${kubernetes.pod.ip}` and `${kubernetes.pod.labels.app}`

#### High level description
The Kubernetes dynamic provider watches for Kubernetes resources and generates mappings from them (similar to events in beats provider). The mappings include those variables([list of variables](https://www.elastic.co/guide/en/fleet/03bf16907bea9768427f8305a5c345368b55d834/dynamic-input-configuration.html#kubernetes-provider)) for each k8s resource with unique value for each one of them.
Agent composable controller which controls all the providers receives these mappings and tries to match them with the  input blogs of the configurations.
This means that for every mapping that the condition matches (kubernetes.pod.labels.app equals to redis), a
new input will be created in which the condition will be removed(not needed anymore) and the `kubernetes.pod.ip` variable will be substituted from the value in the same mapping.
The updated complete inputs blog will be then forwarded to agent to spawn/update metricbeat and filebeat instances.

##### Internals

Step-by-step walkthrough
1. Elastic agent running in local mode initiates a new [composable controller](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/x-pack/elastic-agent/pkg/agent/application/local_mode.go#L112).
2. The controller consists of all contextProviders and [dynamicProviders](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/x-pack/elastic-agent/pkg/composable/controller.go#L73).
3. Agent initiates a new [emitter](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/x-pack/elastic-agent/pkg/agent/application/local_mode.go#L118) which [starts](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/x-pack/elastic-agent/pkg/agent/application/pipeline/emitter/emitter.go#L27) all the dynamicProviders of the [controller](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/x-pack/elastic-agent/pkg/composable/controller.go#L122).
4. Kubernetes Dynamic provider depending on the [resource type](https://github.com/elastic/beats/blob/03bf16907bea9768427f8305a5c345368b55d834/x-pack/elastic-agent/pkg/composable/providers/kubernetes/kubernetes.go#L56) (default is pod) initiates a [watcher](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/providers/kubernetes/pod.go#L69) for
   the specific resource the same way as in metrcbeat/filebeat kubernetes provider.
5. Under the hood a dedicated watcher starts for pods, nodes and namespaces as well as a metadata generator.
6. The difference is that the watchers instead of publishing events, they create [data](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/providers/kubernetes/pod.go#L134) from the objects read from the queue. These [data](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/providers/kubernetes/pod.go#L244) consist of mappings, processors and a priority .
7. A [mapping](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/providers/kubernetes/pod.go#L217) includes all those variables retrieved from the kubernetes resource metadata while the [processors](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/providers/kubernetes/pod.go#L236) indicate the addition of extra fields.
8. Composable controller [collects](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/controller.go#L244) the created data and [compares](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/controller.go#L263) their mappings and processors against the existing ones. If there is a change, it updates the dynamicProviderState and notifies a worker thread through a [signal](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/controller.go#L272).
9. When the worker gets [notified](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/controller.go#L141) for a change it creates new [variables](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/composable/controller.go#L170) from the mappings and processors.
10. It then updates the [emitter](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/agent/application/pipeline/emitter/controller.go#L111) with them.
11. The emitter controller will update the ast that is then used by the agent to generate the final inputs and spawn new programs to deploy the changes([code](https://github.com/elastic/beats/blob/3c77c9a92a2e90b85f525293cb4c2cfc5bc996b1/x-pack/elastic-agent/pkg/agent/application/pipeline/emitter/controller.go#L151)).
