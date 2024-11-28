// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	// Register the ModuleFactory function for the "azure" module.
	if err := mb.Registry.AddModule("azure", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// azure module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	return &base, nil
}

// MetricStore holds the accumulated metric definitions with a mutex for synchronization.
type MetricStore struct {
	sync.Mutex
	accumulatedMetrics []Metric
}

func (s *MetricStore) AddMetric(metric Metric) {
	s.Lock()         // Acquire the lock
	defer s.Unlock() // Ensure the lock is released when the function returns
	s.accumulatedMetrics = append(s.accumulatedMetrics, metric)
}

func (s *MetricStore) GetMetrics() []Metric {
	s.Lock()         // Acquire the lock to prevent writes while reading
	defer s.Unlock() // Ensure the lock is released when the function returns
	return s.accumulatedMetrics
}

// ClearMetrics clears all accumulated metrics
func (s *MetricStore) ClearMetrics() {
	s.Lock()                          // Acquire the lock
	defer s.Unlock()                  // Ensure the lock is released when the function returns
	s.accumulatedMetrics = []Metric{} // Reset the accumulated metrics slice
}

// Size returns the size of the store
func (s *MetricStore) Size() int {
	return len(s.GetMetrics())
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Client     *Client
	MapMetrics mapResourceMetrics
}

// NewMetricSet will instantiate a new azure metricset
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	metricsetName := base.Name()
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, fmt.Errorf("error unpack raw module config using UnpackConfig: %w", err)
	}

	//validate config based on metricset
	switch metricsetName {
	case nativeMetricset:
		// resources must be configured for the monitor metricset
		if len(config.Resources) == 0 {
			return nil, fmt.Errorf("no resource options defined: module azure - %s metricset", metricsetName)
		}
	default:
		// validate config resource options entered, no resource queries allowed for the compute_vm and compute_vm_scaleset metricsets
		for _, resource := range config.Resources {
			if resource.Query != "" {
				return nil, fmt.Errorf("error initializing the monitor client: module azure - %s metricset. No queries allowed, please select one of the allowed options", metricsetName)
			}
		}
		// check for lightweight resources if no groups or ids have been entered, if not a new resource is created to check the entire subscription
		var resources []ResourceConfig
		for _, resource := range config.Resources {
			if hasConfigOptions(resource.Group) || hasConfigOptions(resource.Id) {
				resources = append(resources, resource)
			}
		}
		// check if this is a light metricset or not and no resources have been configured
		if len(resources) == 0 && len(config.Resources) != 0 {
			resources = append(resources, ResourceConfig{
				Query:   fmt.Sprintf("resourceType eq '%s'", config.DefaultResourceType),
				Metrics: config.Resources[0].Metrics,
			})
		}
		config.Resources = resources
	}
	// instantiate monitor client
	monitorClient, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("error initializing the monitor client: module azure - %s metricset: %w", metricsetName, err)
	}
	return &MetricSet{
		BaseMetricSet: base,
		Client:        monitorClient,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right metricset
// It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Set the reference time for the current fetch.
	//
	// The reference time is used to calculate time intervals
	// and compare with collection info in the metric
	// registry to decide whether to collect metrics or not,
	// depending on metric time grain (check `MetricRegistry`
	// for more information).
	//
	// We round the reference time to the nearest second to avoid
	// millisecond variations in the collection period causing
	// skipped collections.
	//
	// See "Round outer limits" and "Round inner limits" tests in
	// the metric_registry_test.go for more information.
	referenceTime := time.Now().UTC()

	// Initialize cloud resources and monitor metrics
	// information.
	//
	// The client collects and stores:
	// - existing cloud resource definitions (e.g. VMs, DBs, etc.)
	// - metric definitions for the resources (e.g. CPU, memory, etc.)
	//
	// The metricset periodically refreshes the information
	// after `RefreshListInterval` (default 600s for
	// most metricsets).
	m.Client.Log.Infof("Fetch called at %s", referenceTime)
	err := m.Client.InitResources(m.MapMetrics)
	if err != nil {
		return err
	}
	// Check if the channel is nil before entering the loop
	if m.Client.ResourceConfigurations.MetricDefinitionsChan == nil {
		return fmt.Errorf("no resources were found based on all the configurations options entered")
	}

	metricStores := make(map[ResDefGroupingCriteria]*MetricStore)

	for {
		select {
		case resMetricDefinition, ok := <-m.Client.ResourceConfigurations.MetricDefinitionsChan:
			if !ok {
				// Data channel closed, stop processing further data
				m.Client.Log.Infof("MetricDefinitionsChan channel closed")
				if len(m.Client.ResourceConfigurations.MetricDefinitionsChan) == 0 {
					m.Client.Log.Debug("no resources were found based on all the configurations options entered")
				}
				m.Client.Log.Infof("processAllStores")
				processAllStores(m.Client, metricStores, referenceTime, report)
				m.Client.Log.Infof("MetricDefinitionsChan is not ok closing")
				m.Client.ResourceConfigurations.MetricDefinitionsChan = nil
			} else {
				// Process each metric definition as it arrives
				m.Client.Log.Infof("MetricDefinitionsChan channel got %+v", resMetricDefinition)
				if len(resMetricDefinition) == 0 {
					return fmt.Errorf("error mapping metrics to events: %w", err)
				}
				resId := resMetricDefinition[0].ResourceId
				m.Client.ResourceConfigurations.Metrics[resId] = resMetricDefinition
				m.Client.GroupAndStoreMetrics(resMetricDefinition, referenceTime, metricStores)
				for criteria, store := range metricStores {
					m.Client.Log.Infof("Store %+v size is %d", criteria, store.Size())
					if store.Size() >= BatchApiResourcesLimit {
						processStore(m.Client, criteria, store, referenceTime, report)
					}
				}
			}
		case err, ok := <-m.Client.ResourceConfigurations.ErrorChan:
			if ok && err != nil {
				// Handle error received from error channel
				return err
			}
			m.Client.Log.Infof("ErrorChan channel closed")
			// Error channel is closed, stop error handling
			m.Client.ResourceConfigurations.ErrorChan = nil
		}

		// Break the loop when both Data and Error channels are closed
		if m.Client.ResourceConfigurations.MetricDefinitionsChan == nil && m.Client.ResourceConfigurations.ErrorChan == nil {
			m.Client.Log.Infof("Both channels closed. breaking")
			break
		}
	}
	processAllStores(m.Client, metricStores, referenceTime, report)
	return nil
}

// hasConfigOptions func will check if any resource id or resource group options have been entered in the light metricsets
func hasConfigOptions(config []string) bool {
	if config == nil {
		return false
	}
	for _, group := range config {
		if group == "" {
			return false
		}
	}
	return true
}
