package events

import (
	"errors"
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
	if err := mb.Registry.AddMetricSet("kubernetes", "events", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// The events MetricSet listens to events from Kubernetes API server and streams them to the output.
// MetricSet implements the mb.PushMetricSet interface, and therefore does not rely on polling.
type MetricSet struct {
	mb.BaseMetricSet
	watcher *EventWatcher
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The kubernetes events metricset is experimental")

	config := defaultKuberentesEventsConfig()

	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes events configuration: %s", err)
	}

	err = validate(config)
	if err != nil {
		return nil, err
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

	watcher := NewEventWatcher(client, config.SyncPeriod, config.Namespace)

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
	event := common.MapStr{
		"firstOccuranceTimestamp": eve.FirstTimestamp.UTC(),
		"lastOccuranceTimestamp":  eve.LastTimestamp.UTC(),
		"message":                 eve.Message,
		"reason":                  eve.Reason,
		"type":                    eve.Type,
		"count":                   eve.Count,
		"metadata":                eve.Metadata,
		"involvedObject":          eve.InvolvedObject,
		"tags": common.MapStr{
			"host": eve.Source.Host,
		},
	}

	if eve.InvolvedObject.Kind == "Pod" {
		event["tags"].(common.MapStr)["pod"] = eve.InvolvedObject.Name
	}

	return event
}

func validate(config kubeEventsConfig) error {
	if !config.InCluster && config.KubeConfig == "" {
		return errors.New("`kube_config` path can't be empty when in_cluster is set to false")
	}
	return nil
}
