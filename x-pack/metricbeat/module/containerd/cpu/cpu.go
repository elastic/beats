// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cpu

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/containerd"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
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
	preSystemCpuUsage          float64
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
		calcPct:                    config.CalculatePct,
		preSystemCpuUsage:          0.0,
		preContainerCpuTotalUsage:  map[string]float64{},
		preContainerCpuKernelUsage: map[string]float64{},
		preContainerCpuUserUsage:   map[string]float64{},
	}, nil
}

// Fetch gathers information from the containerd and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	families, err := m.mod.GetContainerdMetricsFamilies(m.prometheusClient)
	if err != nil {
		return errors.Wrap(err, "error getting families")
	}

	events, err := m.prometheusClient.ProcessMetrics(families, mapping)
	if err != nil {
		return errors.Wrap(err, "error getting events")
	}

	var systemTotalNs int64
	perContainerCpus := make(map[string]int)

	if m.calcPct {
		for _, event := range events {
			systemTotalSeconds, err := event.GetValue("system.total")
			if err == nil {
				systemTotalNs = systemTotalSeconds.(int64) * 1e9
			}
			if _, err = event.GetValue("cpu"); err == nil {
				// calculate cpus used by each container
				setContCpus(event, perContainerCpus)
			}
		}
	}

	for _, event := range events {
		// Discard event containing system.total
		if _, err := event.GetValue("system.total"); err == nil {
			continue
		}
		// setting ECS container.id

		rootFields := common.MapStr{}
		containerFields := common.MapStr{}
		var cID string
		if containerID, ok := event["id"]; ok {
			cID = (containerID).(string)
			containerFields.Put("id", cID)
			event.Delete("id")
		}

		if len(containerFields) > 0 {
			rootFields.Put("container", containerFields)
		}
		if m.calcPct {
			contCpus, ok := perContainerCpus[cID]
			if !ok {
				contCpus = 1
			}
			// calculate system usage delta
			systemUsageDelta := float64(systemTotalNs) - m.preSystemCpuUsage

			// Calculate cpu total usage percentage
			cpuUsageTotal, err := event.GetValue("usage.total.ns")
			if err == nil {
				cpuUsageTotalPct := calcUsagePct(cpuUsageTotal.(float64), systemUsageDelta,
					float64(contCpus), cID, m.preContainerCpuTotalUsage)
				m.Logger().Debugf("cpuUsageTotalPct for %+v is %+v", cID, cpuUsageTotalPct)
				event.Put("usage.total.pct", cpuUsageTotalPct)
				// Update values
				m.preContainerCpuTotalUsage[cID] = cpuUsageTotal.(float64)
			}

			// Calculate cpu kernel usage percentage
			cpuUsageKernel, err := event.GetValue("usage.kernel.ns")
			if err == nil {
				cpuUsageKernelPct := calcUsagePct(cpuUsageKernel.(float64), systemUsageDelta,
					float64(contCpus), cID, m.preContainerCpuKernelUsage)
				m.Logger().Debugf("cpuUsageKernelPct for %+v is %+v", cID, cpuUsageKernelPct)
				event.Put("usage.kernel.pct", cpuUsageKernelPct)
				// Update values
				m.preContainerCpuKernelUsage[cID] = cpuUsageKernel.(float64)
			}

			// Calculate cpu user usage percentage
			cpuUsageUser, err := event.GetValue("usage.user.ns")
			if err == nil {
				cpuUsageUserPct := calcUsagePct(cpuUsageUser.(float64), systemUsageDelta,
					float64(contCpus), cID, m.preContainerCpuUserUsage)
				m.Logger().Debugf("cpuUsageUserPct for %+v is %+v", cID, cpuUsageUserPct)
				event.Put("usage.user.pct", cpuUsageUserPct)
				// Update values
				m.preContainerCpuUserUsage[cID] = cpuUsageUser.(float64)
			}
		}
		if cpuId, err := event.GetValue("cpu"); err == nil {
			perCpuNs, err := event.GetValue("usage.percpu.ns")
			if err == nil {
				key := fmt.Sprintf("usage.cpu.%s.ns", cpuId)
				event.Put(key, perCpuNs)
				event.Delete("cpu")
				event.Delete("usage.percpu.ns")
			}
		}

		reporter.Event(mb.Event{
			RootFields:      rootFields,
			MetricSetFields: event,
			Namespace:       "containerd.cpu",
		})
	}
	m.preSystemCpuUsage = float64(systemTotalNs)
	return nil
}

func setContCpus(event common.MapStr, perContainerCpus map[string]int) {
	val, err := event.GetValue("id")
	if err != nil {
		return
	}
	cid := val.(string)
	_, err = event.GetValue("usage.percpu.ns")
	if err != nil {
		return
	}
	if _, ok := perContainerCpus[cid]; ok {
		perContainerCpus[cid] += 1
	} else {
		perContainerCpus[cid] = 1
	}
}

func calcUsagePct(newValue, systemUsageDelta, contCpus float64,
	cid string, oldValuesMap map[string]float64) float64 {
	var usageDelta, usagePct float64
	if oldValue, ok := oldValuesMap[cid]; ok {
		usageDelta = newValue - oldValue
	} else {
		usageDelta = newValue
	}
	if usageDelta == 0.0 || systemUsageDelta == 0.0 {
		usagePct = 0.0
	} else {
		// normalize percentage with cpus used per container
		usagePct = (usageDelta / systemUsageDelta) / contCpus
	}
	return usagePct
}
