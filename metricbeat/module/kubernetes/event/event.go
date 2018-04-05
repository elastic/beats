package event

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/metricbeat/mb"
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
	watcher kubernetes.Watcher
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The kubernetes event metricset is beta")

	config := defaultKubernetesEventsConfig()

	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes event configuration: %s", err)
	}

	client, err := kubernetes.GetKubernetesClient(config.InCluster, config.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("fail to get kubernetes client: %s", err.Error())
	}

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Event{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Namespace:   config.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("fail to init kubernetes watcher: %s", err.Error())
	}

	return &MetricSet{
		BaseMetricSet: base,
		watcher:       watcher,
	}, nil
}

// Run method provides the Kubernetes event watcher with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporter) {
	now := time.Now()
	handler := kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj kubernetes.Resource) {
			reporter.Event(generateMapStrFromEvent(obj.(*kubernetes.Event)))
		},
		UpdateFunc: func(obj kubernetes.Resource) {
			reporter.Event(generateMapStrFromEvent(obj.(*kubernetes.Event)))
		},
		// ignore events that are deleted
		DeleteFunc: nil,
	}
	m.watcher.AddEventHandler(kubernetes.FilteringResourceEventHandler{
		// skip events happened before watch
		FilterFunc: func(obj kubernetes.Resource) bool {
			eve := obj.(*kubernetes.Event)
			if eve.LastTimestamp.Before(now) {
				return false
			}
			return true
		},
		Handler: handler,
	})
	// start event watcher
	m.watcher.Start()
	<-reporter.Done()
	m.watcher.Stop()
	return
}

func generateMapStrFromEvent(eve *kubernetes.Event) common.MapStr {
	eventMeta := common.MapStr{
		"timestamp": common.MapStr{
			"created": eve.Metadata.CreationTimestamp,
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
			safemapstr.Put(labels, k, v)
		}

		eventMeta["labels"] = labels
	}

	if len(eve.Metadata.Annotations) != 0 {
		annotations := make(common.MapStr, len(eve.Metadata.Annotations))
		for k, v := range eve.Metadata.Annotations {
			safemapstr.Put(annotations, k, v)
		}

		eventMeta["annotations"] = annotations
	}

	output := common.MapStr{
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

	tsMap := make(common.MapStr)

	if eve.FirstTimestamp != nil {
		tsMap["first_occurrence"] = eve.FirstTimestamp.UTC()
	}

	if eve.LastTimestamp != nil {
		tsMap["last_occurrence"] = eve.LastTimestamp.UTC()
	}

	if len(tsMap) != 0 {
		output["timestamp"] = tsMap
	}

	return output
}
