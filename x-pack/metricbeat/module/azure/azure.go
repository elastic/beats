// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"slices"
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
	Client         *Client
	MapMetrics     mapResourceMetrics
	BatchClient    *BatchClient
	ConcMapMetrics concurrentMapResourceMetrics // In combination with BatchClient only
}

var supportedMonitorMetricsets = []string{"monitor", "container_registry", "container_instance", "container_service", "compute_vm", "compute_vm_scaleset", "database_account", "storage"}

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
	var monitorClient *Client
	var monitorBatchClient *BatchClient
	// check wether metricset is part of supported metricsets and if BatchApi is enabled
	if slices.Contains(supportedMonitorMetricsets, metricsetName) && config.EnableBatchApi {
		// instantiate Batch Client which enables fetching metric values for multiple resources using azure batch api
		monitorBatchClient, err = NewBatchClient(config)
		if err != nil {
			return nil, fmt.Errorf("error initializing the monitor batch client: module azure - %s metricset: %w", metricsetName, err)
		}
	} else {
		// default case
		monitorClient, err = NewClient(config)
		if err != nil {
			return nil, fmt.Errorf("error initializing the monitor client: module azure - %s metricset: %w", metricsetName, err)
		}
	}

	return &MetricSet{
		BaseMetricSet: base,
		Client:        monitorClient,
		BatchClient:   monitorBatchClient,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right metricset
// It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	if m.BatchClient != nil {
		// EnableBatchApi is true
		return fetchBatch(m, report)
	}
	// default case
	return fetch(m, report)
}

// fetch fetches metric definitions of requested resources, collects the metric values and publishes them
func fetch(m *MetricSet, report mb.ReporterV2) error {
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
	err := m.Client.InitResources(m.MapMetrics)
	if err != nil {
		return err
	}

	if len(m.Client.ResourceConfigurations.Metrics) == 0 {
		// error message is previously logged in the InitResources,
		// no error event should be created
		return nil
	}

	// Group metric definitions by cloud resource ID.
	//
	// We group the metric definitions by resource ID to fetch
	// metric values for each cloud resource in one API call.
	metricsByResourceId := groupMetricsDefinitionsByResourceId(m.Client.ResourceConfigurations.Metrics)

	for _, metricsDefinition := range metricsByResourceId {
		// Fetch metric values for each resource.
		metricValues := m.Client.GetMetricValues(referenceTime, metricsDefinition, report)

		// Turns metric values into events and sends them to Elasticsearch.
		if err := m.Client.MapToEvents(metricValues, report); err != nil {
			return fmt.Errorf("error mapping metrics to events: %w", err)
		}
	}

	return nil
}

// fetchBatch uses concurrency to collect metric definitions of requested resources,
// collects the metrics using the batch Api and publishes them
func fetchBatch(m *MetricSet, report mb.ReporterV2) error {
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
	err := m.BatchClient.InitResources(m.ConcMapMetrics)
	if err != nil {
		return err
	}
	// Check if the channel is nil before entering the loop
	if m.BatchClient.ResourceConfigurations.MetricDefinitionsChan == nil {
		return fmt.Errorf("no resources were found based on all the configurations options entered")
	}

	metricStores := make(map[ResDefGroupingCriteria]*MetricStore)

	for {
		select {
		case resMetricDefinition, ok := <-m.BatchClient.ResourceConfigurations.MetricDefinitionsChan:
			if !ok {
				// Data channel closed, stop processing further data
				m.BatchClient.Log.Debug("MetricDefinitionsChan channel closed")
				if len(m.BatchClient.ResourceConfigurations.MetricDefinitionsChan) == 0 {
					m.BatchClient.Log.Debug("no resources were found based on all the configurations options entered")
				}
				// process all stores in case there are remaining metricstores for which values are not collected.
				m.BatchClient.Log.Debug("processAllStores")
				metricValues := processAllStores(m.BatchClient, metricStores, referenceTime, report)
				if len(metricValues) > 0 {
					if err := m.BatchClient.MapToEvents(metricValues, report); err != nil {
						m.BatchClient.Log.Errorf("error mapping metrics to events: %v", err)
					}
				}
				m.BatchClient.ResourceConfigurations.MetricDefinitionsChan = nil
			} else {
				// Process each metric definition as it arrives
				if len(resMetricDefinition) == 0 {
					return fmt.Errorf("error mapping metrics to events: %w", err)
				}
				if m.BatchClient.ResourceConfigurations.MetricDefinitions.Update {
					// Update MetricDefinitions because they have expired
					resId := resMetricDefinition[0].ResourceId
					m.BatchClient.Log.Debug("MetricDefinitions Data need update")
					m.BatchClient.ResourceConfigurations.MetricDefinitions.Metrics[resId] = resMetricDefinition
				}
				m.BatchClient.GroupAndStoreMetrics(resMetricDefinition, referenceTime, metricStores)
				var metricValues []Metric
				// check if the store size is >= BatchApiResourcesLimit and then process the store(collect metric values)
				for criteria, store := range metricStores {
					if store.Size() >= BatchApiResourcesLimit {
						m.BatchClient.Log.Debugf("Store %+v size is %d. Process the Store", criteria, store.Size())
						metricValues = append(metricValues, processStore(m.BatchClient, criteria, store, referenceTime, report)...)
					}
				}
				// Map the collected metric values into events and publish them.
				if len(metricValues) > 0 {
					if err := m.BatchClient.MapToEvents(metricValues, report); err != nil {
						m.BatchClient.Log.Errorf("error mapping metrics to events: %v", err)
					}
				}
			}
		case err, ok := <-m.BatchClient.ResourceConfigurations.ErrorChan:
			if ok && err != nil {
				// Handle error received from error channel
				return err
			}
			m.BatchClient.Log.Debug("ErrorChan channel closed")
			// Error channel is closed, stop error handling
			m.BatchClient.ResourceConfigurations.ErrorChan = nil
		}

		// Break the loop when both Data and Error channels are closed
		if m.BatchClient.ResourceConfigurations.MetricDefinitionsChan == nil && m.BatchClient.ResourceConfigurations.ErrorChan == nil {
			m.BatchClient.Log.Debug("Both channels closed. breaking")
			break
		}
	}
	// process all stores in case there are remaining metricstores for which values are not collected.
	metricValues := processAllStores(m.BatchClient, metricStores, referenceTime, report)
	if len(metricValues) > 0 {
		if err := m.BatchClient.MapToEvents(metricValues, report); err != nil {
			m.BatchClient.Log.Errorf("error mapping metrics to events: %v", err)
		}
	}
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
