// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package event

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common"
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
	watcher      kubernetes.Watcher
	watchOptions kubernetes.WatchOptions
	dedotConfig  dedotConfig
}

// dedotConfig defines LabelsDedot and AnnotationsDedot.
// If set to true, replace dots in labels with `_`.
// Default to be true.
type dedotConfig struct {
	LabelsDedot      bool `config:"labels.dedot"`
	AnnotationsDedot bool `config:"annotations.dedot"`
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultKubernetesEventsConfig()

	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes event configuration: %s", err)
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("fail to get kubernetes client: %s", err.Error())
	}

	watchOptions := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Namespace:   config.Namespace,
	}

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Event{}, watchOptions)
	if err != nil {
		return nil, fmt.Errorf("fail to init kubernetes watcher: %s", err.Error())
	}

	dedotConfig := dedotConfig{
		LabelsDedot:      config.LabelsDedot,
		AnnotationsDedot: config.AnnotationsDedot,
	}

	return &MetricSet{
		BaseMetricSet: base,
		dedotConfig:   dedotConfig,
		watcher:       watcher,
		watchOptions:  watchOptions,
	}, nil
}

// Run method provides the Kubernetes event watcher with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporter) {
	now := time.Now()
	handler := kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			reporter.Event(generateMapStrFromEvent(obj.(*kubernetes.Event), m.dedotConfig))
		},
		UpdateFunc: func(obj interface{}) {
			reporter.Event(generateMapStrFromEvent(obj.(*kubernetes.Event), m.dedotConfig))
		},
		// ignore events that are deleted
		DeleteFunc: nil,
	}
	m.watcher.AddEventHandler(kubernetes.FilteringResourceEventHandler{
		// skip events happened before watch
		FilterFunc: func(obj interface{}) bool {
			eve := obj.(*kubernetes.Event)
			if kubernetes.Time(&eve.LastTimestamp).Before(now) {
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

func generateMapStrFromEvent(eve *kubernetes.Event, dedotConfig dedotConfig) common.MapStr {
	eventMeta := common.MapStr{
		"timestamp": common.MapStr{
			"created": kubernetes.Time(&eve.ObjectMeta.CreationTimestamp).UTC(),
		},
		"name":             eve.ObjectMeta.GetName(),
		"namespace":        eve.ObjectMeta.GetNamespace(),
		"self_link":        eve.ObjectMeta.GetSelfLink(),
		"generate_name":    eve.ObjectMeta.GetGenerateName(),
		"uid":              eve.ObjectMeta.GetUID(),
		"resource_version": eve.ObjectMeta.GetResourceVersion(),
	}

	if len(eve.ObjectMeta.Labels) != 0 {
		labels := make(common.MapStr, len(eve.ObjectMeta.Labels))
		for k, v := range eve.ObjectMeta.Labels {
			if dedotConfig.LabelsDedot {
				label := common.DeDot(k)
				labels.Put(label, v)
			} else {
				safemapstr.Put(labels, k, v)
			}
		}

		eventMeta["labels"] = labels
	}

	if len(eve.ObjectMeta.Annotations) != 0 {
		annotations := make(common.MapStr, len(eve.ObjectMeta.Annotations))
		for k, v := range eve.ObjectMeta.Annotations {
			if dedotConfig.AnnotationsDedot {
				annotation := common.DeDot(k)
				annotations.Put(annotation, v)
			} else {
				safemapstr.Put(annotations, k, v)
			}
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

	tsMap["first_occurrence"] = kubernetes.Time(&eve.FirstTimestamp).UTC()
	tsMap["last_occurrence"] = kubernetes.Time(&eve.LastTimestamp).UTC()

	if len(tsMap) != 0 {
		output["timestamp"] = tsMap
	}

	return output
}
