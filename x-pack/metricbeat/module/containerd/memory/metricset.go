package memory

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/containerd"

	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/util"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// Metricset for apiserver is a prometheus based metricset
type metricset struct {
	mb.BaseMetricSet
	prometheusClient   prometheus.Prometheus
	prometheusMappings *prometheus.MetricsMapping
	calcPct            bool
}

var _ mb.ReportingMetricSetV2Error = (*metricset)(nil)

// getMetricsetFactory as required by` mb.Registry.MustAddMetricSet`
func getMetricsetFactory(prometheusMappings *prometheus.MetricsMapping) mb.MetricSetFactory {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		pc, err := prometheus.NewPrometheusClient(base)
		if err != nil {
			return nil, err
		}
		config := containerd.DefaultConfig()
		if err := base.Module().UnpackConfig(&config); err != nil {
			return nil, err
		}
		return &metricset{
			BaseMetricSet:      base,
			prometheusClient:   pc,
			prometheusMappings: prometheusMappings,
			calcPct:            config.CalculatePct,
		}, nil
	}
}

// Fetch gathers information from the containerd and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	events, err := m.prometheusClient.GetProcessedMetrics(m.prometheusMappings)
	if err != nil {
		return errors.Wrap(err, "error getting metrics")
	}

	for _, event := range events {

		// setting ECS container.id
		containerFields := common.MapStr{}
		var cID string
		if containerID, ok := event["id"]; ok {
			cID = (containerID).(string)
			containerFields.Put("id", cID)
			event.Delete("id")
		}
		e, err := util.CreateEvent(event, "containerd.memory")
		if err != nil {
			m.Logger().Error(err)
		}

		if len(containerFields) > 0 {
			if e.RootFields != nil {
				e.RootFields.DeepUpdate(common.MapStr{
					"container": containerFields,
				})
			} else {
				e.RootFields = common.MapStr{
					"container": containerFields,
				}
			}
		}

		// Calculate memory total usage percentage
		if m.calcPct {
			inactiveFiles, err := event.GetValue("inactiveFiles")
			if err == nil {
				usageTotal, err := event.GetValue("usage.total")
				if err == nil {
					memoryLimit, err := event.GetValue("usage.limit")
					if err == nil {
						usage := usageTotal.(float64) - inactiveFiles.(float64)
						memoryUsagePct := usage / memoryLimit.(float64)
						e.MetricSetFields.Put("usage.pct", memoryUsagePct)
						m.Logger().Debugf("memoryUsagePct for %+v is %+v", cID, memoryUsagePct)
					}
				}
			}
		}

		if reported := reporter.Event(e); !reported {
			return nil
		}
	}
	return nil
}
