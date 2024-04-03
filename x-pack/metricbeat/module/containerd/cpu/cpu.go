// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cpu

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/containerd"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/v1/metrics"
)

var (
	// HostParser validates Prometheus URLs
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "metrics_path",
	}.Build()

	// Mapping of state metrics
	mapping = &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"container_cpu_total_nanoseconds":  prometheus.Metric("usage.total.ns"),
			"container_cpu_user_nanoseconds":   prometheus.Metric("usage.user.ns"),
			"container_cpu_kernel_nanoseconds": prometheus.Metric("usage.kernel.ns"),
			"container_per_cpu_nanoseconds":    prometheus.Metric("usage.percpu.ns"),
			"process_cpu_seconds_total":        prometheus.Metric("system.total"),
		},
		Labels: map[string]prometheus.LabelMap{
			"container_id": prometheus.KeyLabel("id"),
			"namespace":    prometheus.KeyLabel("namespace"),
			"cpu":          prometheus.KeyLabel("cpu"),
		},
	}
)

// Metricset for containerd is a prometheus based metricset
type metricset struct {
	mb.BaseMetricSet
	prometheusClient           prometheus.Prometheus
	mod                        containerd.Module
	calcPct                    bool
	preTimestamp               time.Time
	preContainerCpuTotalUsage  map[string]float64
	preContainerCpuKernelUsage map[string]float64
	preContainerCpuUserUsage   map[string]float64
}

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("containerd", "cpu", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The containerd cpu metricset is beta.")

	pc, err := prometheus.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}
	config := containerd.DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	mod, ok := base.Module().(containerd.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of kubernetes module")
	}
	return &metricset{
		BaseMetricSet:              base,
		prometheusClient:           pc,
		mod:                        mod,
		calcPct:                    config.CalculateCpuPct,
		preTimestamp:               time.Time{},
		preContainerCpuTotalUsage:  map[string]float64{},
		preContainerCpuKernelUsage: map[string]float64{},
		preContainerCpuUserUsage:   map[string]float64{},
	}, nil
}

func createDimensionsKey(containerId, namespace string) string {
	return fmt.Sprintf("%s-%s", containerId, namespace)
}

// groupByFields aggregates metrics by their common dimensions into consolidated events.
// It creates a map where each key represents a unique pairing of container ID and namespace,
// ensuring that metrics with identical dimensions are grouped into a single event,
// preventing duplicated documents which would get dropped when TSDB is enabled.
func (m *metricset) groupByFields(events []mapstr.M) map[string][]mapstr.M {
	groupedMetrics := make(map[string][]mapstr.M, 0)

	for _, event := range events {
		containerID, err := event.GetValue("id")
		if err != nil {
			continue
		}

		namespace, err := event.GetValue("namespace")
		if err != nil {
			continue
		}

		containerIDStr, ok := containerID.(string)
		if !ok {
			continue
		}

		namespaceStr, ok := namespace.(string)
		if !ok {
			continue
		}

		dimensionKey := createDimensionsKey(containerIDStr, namespaceStr)
		groupedMetrics[dimensionKey] = append(groupedMetrics[dimensionKey], event)
	}

	return groupedMetrics
}

// Fetch gathers information from the containerd and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	families, timestamp, err := m.mod.GetContainerdMetricsFamilies(m.prometheusClient)
	if err != nil {
		return fmt.Errorf("error getting families: %w", err)
	}

	events, err := m.prometheusClient.ProcessMetrics(families, mapping)
	if err != nil {
		return fmt.Errorf("error getting events: %w", err)
	}

	// Group the events by their common dimensions
	grouped := m.groupByFields(events)

	perContainerCpus := make(map[string]int)
	if m.calcPct {
		for _, event := range events {
			if _, err = event.GetValue("cpu"); err == nil {
				// calculate cpus used by each container
				setContCpus(event, perContainerCpus)
			}
		}
	}

	// Iterate through each group and consolidate them into a single event per group
	for _, group := range grouped {
		cID := containerd.GetAndDeleteCid(group[0])
		for _, event := range group {
			// setting ECS container.id and module field containerd.namespace
			containerFields := mapstr.M{}
			if m.calcPct {
				contCpus, ok := perContainerCpus[cID]
				if !ok {
					contCpus = 1
				}
				// calculate timestamp delta
				timestampDelta := int64(0)
				if !m.preTimestamp.IsZero() {
					timestampDelta = timestamp.UnixNano() - m.preTimestamp.UnixNano()
				}
				// Calculate cpu total usage percentage
				cpuUsageTotal, err := event.GetValue("usage.total.ns")
				if err == nil {
					cpuUsageTotalPct := calcUsagePct(timestampDelta, cpuUsageTotal.(float64),
						float64(contCpus), cID, m.preContainerCpuTotalUsage)
					m.Logger().Debugf("cpuUsageTotalPct for %+v is %+v", cID, cpuUsageTotalPct)
					_, _ = event.Put("usage.total.pct", cpuUsageTotalPct)
					// Update container.cpu.usage ECS field
					_, _ = containerFields.Put("cpu.usage", cpuUsageTotalPct)
					// Update values
					m.preContainerCpuTotalUsage[cID], _ = cpuUsageTotal.(float64)
				}

				// Calculate cpu kernel usage percentage
				// If event does not contain usage.kernel.ns skip the calculation (event has only system.total)
				cpuUsageKernel, err := event.GetValue("usage.kernel.ns")
				if err == nil {
					cpuUsageKernelPct := calcUsagePct(timestampDelta, cpuUsageKernel.(float64),
						float64(contCpus), cID, m.preContainerCpuKernelUsage)
					m.Logger().Debugf("cpuUsageKernelPct for %+v is %+v", cID, cpuUsageKernelPct)
					_, _ = event.Put("usage.kernel.pct", cpuUsageKernelPct)
					// Update values
					m.preContainerCpuKernelUsage[cID], _ = cpuUsageKernel.(float64)
				}

				// Calculate cpu user usage percentage
				cpuUsageUser, err := event.GetValue("usage.user.ns")
				if err == nil {
					cpuUsageUserPct := calcUsagePct(timestampDelta, cpuUsageUser.(float64),
						float64(contCpus), cID, m.preContainerCpuUserUsage)
					m.Logger().Debugf("cpuUsageUserPct for %+v is %+v", cID, cpuUsageUserPct)
					_, _ = event.Put("usage.user.pct", cpuUsageUserPct)
					// Update values
					m.preContainerCpuUserUsage[cID], _ = cpuUsageUser.(float64)
				}
			}
			if cpuId, err := event.GetValue("cpu"); err == nil {
				perCpuNs, err := event.GetValue("usage.percpu.ns")
				if err == nil {
					key := fmt.Sprintf("usage.cpu.%s.ns", cpuId)
					_, _ = event.Put(key, perCpuNs)
					_ = event.Delete("cpu")
					_ = event.Delete("usage.percpu.ns")
				}
			}
		}

		containerFields := mapstr.M{}
		moduleFields := mapstr.M{}
		rootFields := mapstr.M{}

		namespace := containerd.GetAndDeleteNamespace(group[0])

		_, _ = containerFields.Put("id", cID)
		_, _ = rootFields.Put("container", containerFields)
		_, _ = moduleFields.Put("namespace", namespace)

		metricsetFields := mapstr.M{}

		for _, event := range group {
			metricsetFields.DeepUpdateNoOverwrite(event)
		}

		ev := mb.Event{
			RootFields:      rootFields,
			ModuleFields:    moduleFields,
			MetricSetFields: metricsetFields,
			Namespace:       "containerd.cpu",
		}
		_ = ev.MetricSetFields.Delete("id")
		_ = ev.MetricSetFields.Delete("namespace")

		reporter.Event(ev)
	}

	// set Timestamp of previous event
	m.preTimestamp = timestamp
	return nil
}

func setContCpus(event mapstr.M, perContainerCpus map[string]int) {
	val, err := event.GetValue("id")
	if err != nil {
		return
	}
	cid, _ := val.(string)
	_, err = event.GetValue("usage.percpu.ns")
	if err != nil {
		return
	}
	perContainerCpus[cid] += 1
}

func calcUsagePct(timestampDelta int64, newValue, contCpus float64,
	cid string, oldValuesMap map[string]float64) float64 {
	var usageDelta, usagePct float64
	if oldValue, ok := oldValuesMap[cid]; ok {
		usageDelta = newValue - oldValue
	} else {
		usageDelta = newValue
	}
	if usageDelta == 0.0 || float64(timestampDelta) == 0.0 {
		usagePct = 0.0
	} else {
		// normalize percentage with cpus used per container
		usagePct = (usageDelta / float64(timestampDelta)) / contCpus
	}
	return usagePct
}
