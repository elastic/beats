# Kubernetes autodiscover provider in Beats and Agent

https://www.elastic.co/guide/en/beats/metricbeat/master/configuration-autodiscover.html
https://www.elastic.co/guide/en/beats/filebeat/current/configuration-autodiscover.html
https://www.elastic.co/guide/en/fleet/7.x/dynamic-input-configuration.html#providers

When you run applications on containers, they become moving targets to the monitoring system. Autodiscover allows you to track them and adapt settings as changes happen. By defining configuration templates, the autodiscover subsystem can monitor services as they start running.

You define autodiscover settings in the autodiscover section of the metricbeat.yml/filebeat.yml config file. To enable autodiscover, you specify a list of providers.
For Elastic Agent we will discuss later.

**Providers**
Autodiscover providers work by watching for events on the system and translating those events into internal autodiscover events with a common format. When you configure the provider, you can optionally use fields from the autodiscover event to set conditions that, when met, launch specific configurations.

On start Metricbeat/Filebeat/Agent will scan existing containers and launch the proper configs for them. Then it will watch for new start/stop events. This ensures you don’t need to worry about state, but only define your desired configs.

The Kubernetes autodiscover provider watches for Kubernetes nodes, pods, services and namespaces when they start, update, and stop.

We will describe the internals of three ways of Kubernetes autodiscover process.
1. Templates Based Autodiscover
2. Hints Based Autodiscover
3. Autodiscover provider in Elastic Agent

## Templates Based Autodiscover

As the name suggests, user needs to set a template to indicate to autodiscover provider what to do.
There is one configuration variable that differentiates in a way how the autosicover process is performed.
This variable is `unique`

### Autodiscover with LeaderElection
When setting `unique: true` the Leader Election mechanism is activated. That way **only** the Beat instance that will gain the leader lease/lock will enable the provided template.
The best appliance of this feature is when collecting cluster wide metrics from `kube-state-metrics` or `apiserver`.
In that case having all instances of metricbeat collecting the same metrics is not desirable.

Example:
```
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

##### How it works

We will deep dive in the internals of [libbeat kubernetes autodiscover provider](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes).

We will use metricbeat as an example.

Step-by-step walkthrough
1. Kubernetes provider `init` function [adds](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L46) the provider in the autodiscover providers registry at startup. For Kubernetes provider an `AutodiscoverBuilder` func is passed as an argument.
2. Metricbeat calls `NewAutodiscover` [function](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/metricbeat/beater/metricbeat.go#L183) which checks in the config for enabled providers and [builds](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/provider.go#L90) them one by one, calling the `AutodiscoverBuilder` func.
3. Kubernetes `AutodiscoverBuilder` creates and returns a [Kubernetes Provider struct](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L131) which is then added to an Autodiscover manager struct.
4. When unique is set to true [NewLeaderElectionManager](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L141) is set as the eventManager of Kubernetes Provider.
5. Metricbeat [starts](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/metricbeat/beater/metricbeat.go#L249) the Autodiscover manager which starts for Kubernets provider the [leaderElectionManager](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L326). Before starting the providers it also starts a worker for listening of events that will be published by the eventers of each provider.
6. `OnStartedLeading` is executed when the specific metricbeat instance gains the leader election lock. [StartLeading](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L205) creates a bus event with `"start":    true,`  and publishes it. The template configurations is also added in this event.
7. [Listener](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/autodiscover.go#L140) of events get the published event and generates [configs](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/autodiscover.go#L185) for it. Configs include the variables and settings from the template set by the user.
8. For each config the worker checks if it already [exists](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/autodiscover.go#L221) (It was already handled). If at least one of the events config does not exist, then the config is marked as `updated`.
9. The runners list get [reloaded](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/cfgfile/list.go#L54). It is checked from the list of current runners if each config is handled by one of them.
   If no runner is handling that config a new runner will [start](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/cfgfile/list.go#L107). If some runners are no longer needed will be removed.
10. Starting of a runner starts under the hood the metricbeat [module](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/metricbeat/mb/module/runner.go#L76) with the specific metricsets as they are set in the template. Same also applies for filebeat and heartbeat cases.

### Autodiscover without LeaderElection

When setting `unique: false` or not setting it at all the Leader Election mechanism is disabled. That way all Beat instances will enable the provided template. Or at least the ones that match a condition.
A good appliance of this is when user wants to enable a specific module(for example redis) each time a pod with a specific label appears in each kubernetes node.

Example:
```
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

##### How it works

We will use metricbeat and redis module as example.
Step-by-step walkthrough

Steps 1-3 are exactly the same as with Leader Election.
4. In the [Kubernetes Provider struct](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L131) created by the Kubernetes AutodiscoverBuilder in case there is a [condition](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L116) a `NewConfigMapper` is created. It contains the condition [map](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/template/config.go#L64) with all the conditions of a given config.
4. When unique is set to false, the resource is [checked](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L239) from the configuration template(default is pod) and a new PodEventer/NodeEventer/ServiceEventer is set as the eventManager of Kubernetes Provider.
5. The kubernetes node that the metricbeat instance is running on is [discovered](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L72).
6. A dedicated watcher to watch [pods](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L82), nodes and namespaces get started. Also a [metadata generator](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L113) is started to enrich the events with kubernetes metadata.
7. Each time a new event is about to be published by the watchers(we will have a dedicated section of how events are published), it is checked whether the condition set by the user [matches](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L182) the event.
8. If it matches, then a config is created with the configuration set by user and it is added in the event. After that the event gets published.
9. The worker listening for events receives the start event and checks if there is any update in the configs of the event. (same process with leader election steps 6-9)
10. If there is an update it starts or stops the runners needed. In our example it starts a new runner that starts the redis module.

##### How pod eventer works

The [watcher](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/common/kubernetes/watcher.go#L124) struct has a kubernetes informer field which depending on the resource type  [watches](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/common/kubernetes/informer.go#L52) and lists the changes in the kubernetes cluster for that specific resource.
When a new pod for example is discovered it is added in the work [queue](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/common/kubernetes/watcher.go#L137) for processing.
The watcher processes that [queue](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/common/kubernetes/watcher.go#L218) and depending if there was an addition,deletion or update of a resource it calls the
related [method](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/common/kubernetes/watcher.go#L250).
* [OnAdd](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L137) is called when there is a pod addition and it executes [emit](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L314) with `start` flag. Emit adds the pod to a new event together will all metadata, sets `start: true` and [publishes](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L343) it. Step 7 follows.
* [OnDelete](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L160) is called when there is a pod deletion and it ecexutes emit with `stop` flag. Emit adds the pod to a new event with ``stop: true`` and publishes it. Step 7 follows.
* [OnUpdate](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/pod.go#L146) is called when there is update in the an existing resource and it ecexutes at first emit with `stop` flag to generate a stop event and then with `start` flag to generate a start event.


## Hints Based Autodiscover

https://www.elastic.co/guide/en/beats/metricbeat/master/configuration-autodiscover-hints.html

Metricbeat, Filebeat and Heartbeat support autodiscover based on hints from the provider. The hints system looks for hints in Kubernetes Pod annotations which have the prefix co.elastic.metrics. As soon as the container starts, Metricbeat/Filebeat will check if it contains any hints and launch the proper config for it. Hints tell Metricbeat/Filebeat how to get metrics for the given container.

Example metricbeat configuration:
```
metricbeat.autodiscover:
      providers:
        - type: kubernetes
          node: ${NODE_NAME}
          hints.enabled: true
```

Example redis pod with right annotations:
```
apiVersion: v1
kind: Pod
metadata:
  name: redis
  annotations:
    co.elastic.metrics/module: redis
    co.elastic.metrics/metricsets: info
    co.elastic.metrics/hosts: '${data.host}:6379'
spec:
  containers:
  - name: redis
    image: redis:5.0.4
    command:
      - redis-server
      - "/redis-master/redis.conf"
    env:
    - name: MASTER
      value: "true"
    ports:
    - containerPort: 6379
    resources:
      limits:
        cpu: "0.1"
    volumeMounts:
    - mountPath: /redis-master-data
      name: data
    - mountPath: /redis-master
      name: config
  volumes:
    - name: data
      emptyDir: {}
    - name: config
      configMap:
        name: example-redis-config
        items:
        - key: redis-config
          path: redis.conf
```

##### How hints work

Everything works the same as Autodiscover without LeaderElection until step 8.

8. If there is no conditions in the template set by the user, the configs will be generated from [hints](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L186).
9. Wether hints are enabled or not is part of the [Kubernetes Provider struct](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L121) builders field.
10. [GenerateHints](https://github.com/elastic/beats/blob/eff92354db783001880f4bade9f59942fca747ba/libbeat/autodiscover/builder/helper.go#L213) function looks into the event's annotations. A [hints map](https://github.com/elastic/beats/blob/eff92354db783001880f4bade9f59942fca747ba/libbeat/autodiscover/builder/helper.go#L226) is created with all hints and returned.
11. From those hints, configs are [created](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/builder.go#L97) in the same form as in `Autodiscover without LeaderElection` step 8.
    They contain the same information as if they where set explicitly in the metricbeat configureation but actually derive from the pod annotations.
12. Those configs are then [added](https://github.com/elastic/beats/blob/4b1f69923b3f2abbbf1860295fe5dbff7db3d63c/libbeat/autodiscover/providers/kubernetes/kubernetes.go#L197) in the event and gets published.
13. The process after that is same as in `Autodiscover without LeaderElection` step 9 and onward.



##  Autodiscover provider in Elastic Agent
Follow the link [here](https://github.com/elastic/beats/blob/aa264bdf008a9bf309e61744e9ec8c5586593f12/x-pack/elastic-agent/pkg/composable/providers/kubernetes/README.md).
