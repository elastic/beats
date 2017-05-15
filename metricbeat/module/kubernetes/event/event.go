package event

import (
	"fmt"
	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/ericchiang/k8s"
	"github.com/ghodss/yaml"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("kubernetes", "event", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// The event MetricSet listens to events from Kubernetes API server and streams them to the output.
// MetricSet implements the mb.PushMetricSet interface, and therefore does not rely on polling.
type MetricSet struct {
	mb.BaseMetricSet
	watcher *Watcher
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Experimental("The kubernetes event metricset is experimental")

	config := defaultKuberentesEventsConfig()

	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes event configuration: %s", err)
	}

	var client *k8s.Client
	if config.InCluster == true {
		client, err = k8s.NewInClusterClient()
		if err != nil {
			return nil, fmt.Errorf("Unable to get in cluster configuration")
		}
	} else {
		data, err := ioutil.ReadFile(config.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("read kubeconfig: %v", err)
		}

		// Unmarshal YAML into a Kubernetes config object.
		var config k8s.Config
		if err = yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("unmarshal kubeconfig: %v", err)
		}
		client, err = k8s.NewClient(&config)
		if err != nil {
			return nil, err
		}
	}

	watcher := NewWatcher(client, config.SyncPeriod, config.Namespace)

	return &MetricSet{
		BaseMetricSet: base,
		watcher:       watcher,
	}, nil
}

// Run method provides the Kubernetes event watcher with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporter) {
	// Start event watcher
	m.watcher.Run()

	for {
		select {
		case <-reporter.Done():
			m.watcher.Stop()
			return
		case msg := <-m.watcher.eventQueue:
			// Ignore events that are deleted
			if msg.Metadata.DeletionTimestamp == "" {
				if msg.Metadata.DeletionTimestamp == "" {
					reporter.Event(generateMapStrFromEvent(msg))
				}
			}
		}
	}
}

func generateMapStrFromEvent(eve *Event) common.MapStr {
	eventMeta := common.MapStr{
		"timestamp": common.MapStr{
			"created": eve.Metadata.CreationTimestamp,
			"deleted": eve.Metadata.DeletionTimestamp,
		},
		"name":             eve.Metadata.Name,
		"namespace":        eve.Metadata.Namespace,
		"self_link":        eve.Metadata.SelfLink,
		"generate_name":    eve.Metadata.GenerateName,
		"uid":              eve.Metadata.UID,
		"resource_version": eve.Metadata.ResourceVersion,
	}

	if len(eve.Metadata.Labels) != 0 {
		labels := make(common.MapStr, len(eve.Metadata.Labels))
		for k, v := range eve.Metadata.Labels {
			labels[k] = v
		}

		eventMeta["labels"] = labels
	}

	if len(eve.Metadata.Annotations) != 0 {
		annotations := make(common.MapStr, len(eve.Metadata.Annotations))
		for k, v := range eve.Metadata.Annotations {
			annotations[k] = v
		}

		eventMeta["annotations"] = annotations
	}

	return common.MapStr{
		"timestamp": common.MapStr{
			"first_occurrence": eve.FirstTimestamp.UTC(),
			"last_occurrence":  eve.LastTimestamp.UTC(),
		},
		"message": eve.Message,
		"reason":  eve.Reason,
		"type":    eve.Type,
		"count":   eve.Count,
		"involved_object": common.MapStr{
			"api_version":      eve.InvolvedObject.APIVersion,
			"resource_version": eve.InvolvedObject.ResourceVersion,
			"name":             eve.InvolvedObject.Name,
			"kind":             eve.InvolvedObject.Kind,
			"uid":              eve.InvolvedObject.UID,
		},
		"metadata": eventMeta,
	}

}
