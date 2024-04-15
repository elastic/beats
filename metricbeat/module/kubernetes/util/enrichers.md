## Kubernetes Metadata enrichment

The metadata enrichment process involves associating contextual information, such as Kubernetes metadata (e.g., labels, annotations, resource names), with metrics and events collected by Elastic Agent and Beats in Kubernetes environments. This process enhances the understanding and analysis of collected data by providing additional context.

### Key Components:

1. **Metricsets/Datasets:**
   - Metricsets/Datasets are responsible for collecting metrics and events from various sources within Kubernetes, such as kubelet and kube-state-metrics.

2. **Enrichers:**
   - Enrichers are components responsible for enriching collected data with Kubernetes metadata. Each metricset is associated with its enricher, which handles the metadata enrichment process.

3. **Watchers:**
   - Watchers are mechanisms used to monitor Kubernetes resources and detect changes, such as the addition, update, or deletion of resources like pods or nodes.

4. **Metadata Generators:**
   - Metadata generators are responsible for generating metadata associated with Kubernetes resources. These generators are utilized by enrichers to collect relevant metadata. Each enricher has one metadata generator.

### Metadata Generation Process:

1. **Initialization:**
   - Metricsets are initialized with their respective enrichers during startup. Enrichers are responsible for managing the metadata enrichment process for their associated metricsets.

2. **Watcher Creation:**
   - Multiple enrichers are associated with one watcher. For example a pod watcher is associated with pod, state_pod, container and state_container metricsets and their enrichers. 
   - Watchers are created to monitor Kubernetes resources relevant to the metricset's data collection requirements. For example pod metricset triggers the creation of watcher for pods, nodes and namespaces.

3. **Metadata Generation:**
   - When a watcher detects a change in a monitored resource (e.g., a new pod creation or a label update), it triggers the associated enrichers' metadata generation process.

4. **Enrichment Generation Process:**
   - The enricher collects relevant metadata from Kubernetes API objects corresponding to the detected changes. This metadata includes information like labels, annotations, resource names, etc.

5. **Association with Events:**
   - The collected metadata is then associated with the metricset's events. This association enriches the events with contextual information, providing deeper insights into the collected data. The enriched events generated from beats/agent are then sent to the configured output (e.g. Elasticsearch).

### Handling Edge Cases:

1. **Synchronization:**
   - Special mechanisms are in place to handle scenarios where resources trigger events before associated enrichers are fully initialized. Proactive synchronization ensures that existing resource metadata is captured and updated in enricher maps.
   - When a watcher detects events (like object additions or updates), it updates a list (metadataObjects) with the IDs of these detected objects. Before introducing new enrichers, existing metadataObjects are reviewed. For each existing object ID, the corresponding metadata is retrieved and used to update the new enrichers, ensuring that metadata for pre-existing resources is properly captured and integrated into the new enricher's metadata map. This synchronization process guarantees accurate metadata enrichment, even for resources that triggered events before the initialization of certain enrichers.

### Watcher Management:

1. **Initialization Sequence:**
   - Watchers are initialized and managed by metricsets. Extra watchers, such as those for namespaces and nodes, are initialized first to ensure metadata availability before the main watcher starts monitoring resources.

2. **Configuration Updates:**
   - Watcher configurations, such as watch options or resource filtering criteria, can be updated dynamically. A mechanism is in place to seamlessly transition to updated configurations without disrupting data collection.



In the following diagram, an example of different metricsets leveraging the same watchers is depicted. Metricsets have their own enrichers but share watchers. The watchers monitor the Kubernetes API for specific resource updates.
[metadata diag](../_meta/images/enrichers.png)

### Expected watchers per metricset

The following table demonstrates which watchers are needed for each metricset by default.
Note that no watcher monitoring the same resource kind will be created twice.

| Metricset            | Namespace watcher | Node watcher | Resource watcher | Notes                                                     |
|----------------------|:-----------------:|:------------:|:----------------:|-----------------------------------------------------------|
| API Server           |     &#10005;      |   &#10005;   |     &#10005;     |                                                           |
| Container            |      &check;      |   &check;    |     &check;      |                                                           |
| Controller manager   |     &#10005;      |   &#10005;   |     &check;      |                                                           |
| Event                |     &check;      |   &#10005;   |     &check;      |                                                           |
| Node                 |     &#10005;      |   &check;    |     &check;      | Resource watcher should be the same as node watcher.      |
| Pod                  |      &check;      |   &check;    |     &check;      |                                                           |
| Proxy                |     &#10005;      |   &#10005;   |     &#10005;     |                                                           |
| Scheduler            |     &#10005;      |   &#10005;   |     &#10005;     |                                                           |
| State container      |      &check;      |   &check;    |     &check;      |                                                           |
| State cronjob        |      &check;      |   &#10005;   |     &check;      |                                                           |
| State daemonset      |      &check;      |   &#10005;   |     &check;      |                                                           |
| State deployment     |      &check;      |   &#10005;   |     &check;      |                                                           |
| State job            |      &check;      |   &#10005;    |     &check;      |                                                           |
| State namespace      |      &check;      |   &#10005;   |     &check;      | Resource watcher should be the same as namespace watcher. |
| State node           |     &#10005;      |   &check;    |     &check;      | Resource watcher should be the same as node watcher.      |
| State PV             |     &#10005;      |   &#10005;   |     &check;      |                                                           |
| State PVC            |      &check;      |   &#10005;   |     &check;      |                                                           |
| State pod            |      &check;      |   &check;    |     &check;      |                                                           |
| State replicaset     |      &check;      |   &#10005;   |     &check;      |                                                           |
| State resource quota |     &check;      |   &#10005;   |     &check;     |                                                           |
| State service        |      &check;      |   &#10005;   |     &check;      |                                                           |
| State statefulset    |      &check;      |   &#10005;   |     &check;      |                                                           |
| State storage class  |     &#10005;      |   &#10005;   |     &check;      |                                                           |
| System               |     &#10005;      |   &#10005;   |     &#10005;     |                                                           |
| Volume               |     &#10005;      |   &#10005;   |     &#10005;     |                                                           |

