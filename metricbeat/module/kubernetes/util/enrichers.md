## Kubernetes Metadata enrichment

[metadata diag](../_meta/images/enrichers.png)

The metadata enrichment process involves associating contextual information, such as Kubernetes metadata (e.g., labels, annotations, resource names), with metrics and events collected by Elastic Agent and Beats in Kubernetes environments. This process enhances the understanding and analysis of collected data by providing additional context.

### Key Components:

1. **Metricsets:**
   - Metricsets are responsible for collecting metrics and events from various sources within Kubernetes, such as kubelet and kube-state-metrics.

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
   - Watchers are created to monitor Kubernetes resources relevant to the metricset's data collection requirements. For example pod metricset triggers the creation of watcher for pods, nodes and namespaces.
   - Multiple enrichers are associated with one watcher. For example a pod watcher is associated with pod, state_pod, container and state_container metricsets and their enrichers. 

3. **Metadata Generation:**
   - When a watcher detects a change in a monitored resource (e.g., a new pod creation or a label update), it triggers the associated enrichers' metadata generation process.

4. **Enrichment:**
   - The enricher collects relevant metadata from Kubernetes API objects corresponding to the detected changes. This metadata includes information like labels, annotations, resource names, etc.

5. **Association with Events:**
   - The collected metadata is then associated with the metricset's events. This association enriches the events with contextual information, providing deeper insights into the collected data.

### Handling Edge Cases:

1. **Synchronization:**
   - Special mechanisms are in place to handle scenarios where resources trigger events before associated enrichers are fully initialized. Proactive synchronization ensures that existing resource metadata is captured and updated in enricher maps.

### Watcher Management:

1. **Initialization Sequence:**
   - Watchers are initialized and managed by metricsets. Extra watchers, such as those for namespaces and nodes, are initialized first to ensure metadata availability before the main watcher starts monitoring resources.

2. **Configuration Updates:**
   - Watcher configurations, such as watch options or resource filtering criteria, can be updated dynamically. A mechanism is in place to seamlessly transition to updated configurations without disrupting data collection.

