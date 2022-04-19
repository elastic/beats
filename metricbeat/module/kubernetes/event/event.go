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

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("kubernetes", "event", New)
}

// MetricSet type defines all fields of the MetricSet
// The event MetricSet listens to events from Kubernetes API server and streams them to the output.
// MetricSet implements the mb.PushMetricSet interface, and therefore does not rely on polling.
type MetricSet struct {
	mb.BaseMetricSet
	watcher      kubernetes.Watcher
	watchOptions kubernetes.WatchOptions
	dedotConfig  dedotConfig
	skipOlder    bool
	clusterMeta  common.MapStr
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
		return nil, fmt.Errorf("fail to unpack the kubernetes event configuration: %w", err)
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		return nil, fmt.Errorf("fail to get kubernetes client: %w", err)
	}

	watchOptions := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Namespace:   config.Namespace,
	}

	watcher, err := kubernetes.NewNamedWatcher("event", client, &kubernetes.Event{}, watchOptions, nil)
	if err != nil {
		return nil, fmt.Errorf("fail to init kubernetes watcher: %w", err)
	}

	dedotConfig := dedotConfig{
		LabelsDedot:      config.LabelsDedot,
		AnnotationsDedot: config.AnnotationsDedot,
	}

	ms := &MetricSet{
		BaseMetricSet: base,
		dedotConfig:   dedotConfig,
		watcher:       watcher,
		watchOptions:  watchOptions,
		skipOlder:     config.SkipOlder,
	}

	// add ECS orchestrator fields
	cfg, _ := common.NewConfigFrom(&config)
	ecsClusterMeta, err := getClusterECSMeta(cfg, client)
	if ecsClusterMeta != nil {
		ms.clusterMeta = ecsClusterMeta
	}

	return ms, nil
}

func getClusterECSMeta(cfg *common.Config, client k8sclient.Interface) (common.MapStr, error) {
	clusterInfo, err := metadata.GetKubernetesClusterIdentifier(cfg, client)
	if err != nil {
		return nil, fmt.Errorf("fail to init kubernetes watcher: %w", err)
	}
	ecsClusterMeta := common.MapStr{}
	if clusterInfo.Url != "" {
		ecsClusterMeta.Put("orchestrator.cluster.url", clusterInfo.Url)
	}
	if clusterInfo.Name != "" {
		ecsClusterMeta.Put("orchestrator.cluster.name", clusterInfo.Name)
	}
	return ecsClusterMeta, nil
}

// Run method provides the Kubernetes event watcher with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	now := time.Now()
	handler := kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			mapStrEvent := generateMapStrFromEvent(obj.(*kubernetes.Event), m.dedotConfig, m.Logger())
			event := mb.TransformMapStrToEvent("kubernetes", mapStrEvent, nil)
			if m.clusterMeta != nil {
				event.RootFields.DeepUpdate(m.clusterMeta)
			}
			reporter.Event(event)
		},
		UpdateFunc: func(obj interface{}) {
			mapStrEvent := generateMapStrFromEvent(obj.(*kubernetes.Event), m.dedotConfig, m.Logger())
			event := mb.TransformMapStrToEvent("kubernetes", mapStrEvent, nil)
			if m.clusterMeta != nil {
				event.RootFields.DeepUpdate(m.clusterMeta)
			}
			reporter.Event(event)
		},
		// ignore events that are deleted
		DeleteFunc: nil,
	}
	m.watcher.AddEventHandler(kubernetes.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			eve, ok := obj.(*kubernetes.Event)
			if !ok {
				m.Logger().Debugf("Error while casting event: %s", ok)
			}
			// if fields are null they are decoded to `0001-01-01 00:00:00 +0000 UTC`
			// so we need to check if they are valid first
			lastTimestampValid := !kubernetes.Time(&eve.LastTimestamp).IsZero()
			eventTimeValid := !kubernetes.MicroTime(&eve.EventTime).IsZero()
			// if skipOlder, skip events happened before watch
			if m.skipOlder && kubernetes.Time(&eve.LastTimestamp).Before(now) && lastTimestampValid {
				return false
			} else if m.skipOlder && kubernetes.MicroTime(&eve.EventTime).Before(now) && eventTimeValid {
				// there might be cases that `LastTimestamp` is not a valid number so double check
				// with `EventTime`
				return false
			}
			return true
		},
		Handler: handler,
	})
	// start event watcher
	err := m.watcher.Start()
	if err != nil {
		m.Logger().Debugf("Unable to start watcher: %w", err)
	}
	<-reporter.Done()
	m.watcher.Stop()
}

func generateMapStrFromEvent(eve *kubernetes.Event, dedotConfig dedotConfig, logger *logp.Logger) common.MapStr {
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
				util.ShouldPut(labels, label, v, logger)

			} else {
				err := safemapstr.Put(labels, k, v)
				if err != nil {
					logger.Debugf("Failed to put field '%s' with value '%s': %s", k, v, err)
				}
			}
		}

		eventMeta["labels"] = labels
	}

	if len(eve.ObjectMeta.Annotations) != 0 {
		annotations := make(common.MapStr, len(eve.ObjectMeta.Annotations))
		for k, v := range eve.ObjectMeta.Annotations {
			if dedotConfig.AnnotationsDedot {
				annotation := common.DeDot(k)
				util.ShouldPut(annotations, annotation, v, logger)
			} else {
				err := safemapstr.Put(annotations, k, v)
				if err != nil {
					logger.Debugf("Failed to put field '%s' with value '%s': %s", k, v, err)
				}
			}
		}

		eventMeta["annotations"] = annotations
	}

	output := common.MapStr{
		"message": eve.Message,
		"reason":  eve.Reason,
		"type":    eve.Type,
		"count":   eve.Count,
		"source": common.MapStr{
			"host":      eve.Source.Host,
			"component": eve.Source.Component,
		},
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
