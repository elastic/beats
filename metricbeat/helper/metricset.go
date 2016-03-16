package helper

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Metric specific data
// This must be defined by each metric
type MetricSet struct {
	Name        string
	MetricSeter MetricSeter
	// Inherits config from module
	Config ModuleConfig
	Module *Module
}

// Creates a new MetricSet
func NewMetricSet(name string, new func() MetricSeter, module *Module) (*MetricSet, error) {
	metricSeter := new()

	ms := &MetricSet{
		Name:        name,
		MetricSeter: metricSeter,
		Config:      module.Config,
		Module:      module,
	}

	return ms, nil
}

func (m *MetricSet) Setup() error {

	// In case no hosts are set, set at least an empty string
	// This ensure that also for metricsets where host is not used
	// That fetch is at least called once
	if len(m.Config.Hosts) == 0 {
		m.Config.Hosts = append(m.Config.Hosts, "")
	}

	// Host is a first class citizen and does not have to be handled by the metricset itself
	return m.MetricSeter.Setup(m)
}

// RunMetric runs the given metricSet and returns the event
func (m *MetricSet) Fetch() error {

	for _, host := range m.Config.Hosts {
		// TODO Improve fetching in go routing -> how are go routines stopped if they take too long? - @ruflin,20160314
		m.Module.wg.Add(1)
		go func(h string) {
			defer m.Module.wg.Done()

			event, err := m.MetricSeter.Fetch(m, h)

			if err != nil {
				event["error"] = err
			}
			event = m.createEvent(event)
			m.Module.Publish <- event
		}(host)
	}
	return nil
}

func (m *MetricSet) createEvent(event common.MapStr) common.MapStr {

	timestamp := common.Time(time.Now())

	// Default name is empty, means it will be metricbeat
	indexName := ""
	typeName := "metricsets"

	// Set meta information dynamic if set
	indexName = getIndex(event, indexName)
	typeName = getType(event, typeName)
	timestamp = getTimestamp(event, timestamp)

	// Each metricset has a unique eventfieldname to prevent type conflicts
	eventFieldName := m.Module.name + "-" + m.Name

	event = applySelector(event, m.Config.Selectors)

	// TODO: Add fields_under_root option for "metrics"?
	event = common.MapStr{
		"type":                  typeName,
		eventFieldName:          event,
		"metricset":             m.Name,
		"module":                m.Module.name,
		"@timestamp":            timestamp,
		common.EventMetadataKey: m.Config.EventMetadata,
	}

	// Overwrite index in case it is set
	if indexName != "" {
		event["beat"] = common.MapStr{
			"index": indexName,
		}
	}

	return event
}

func applySelector(event common.MapStr, selectors []string) common.MapStr {

	// No selectors set means return full events
	if len(selectors) == 0 {
		return event
	}

	newEvent := common.MapStr{}
	logp.Debug("metricset", "Applying selectors: %v", selectors)

	for _, selector := range selectors {

		if value, ok := event[selector]; ok {
			newEvent[selector] = value
		}

	}

	return newEvent
}
