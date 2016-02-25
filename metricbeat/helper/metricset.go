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
}

// Creates a new MetricSet
func NewMetricSet(name string, metricset MetricSeter, config ModuleConfig) *MetricSet {
	return &MetricSet{
		Name:        name,
		MetricSeter: metricset,
		Config:      config,
	}
}

// RunMetric runs the given metricSet and returns the event
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	events, err := m.MetricSeter.Fetch(m)

	if err != nil {
		return nil, err
	}
	newEvents := []common.MapStr{}

	// Default names based on module and metric
	// These can be overwritten by setting index or / and type in the event
	indexName := ""

	// Type is the same for all metricsets
	typeName := "metricsets"
	timestamp := common.Time(time.Now())

	for _, event := range events {
		// Set index from event if set
		if _, ok := event["index"]; ok {
			indexName, ok = event["index"].(string)
			if !ok {
				logp.Err("Index couldn't be overwritten because event index is not string")
			}
			delete(event, "index")
		}

		// Set type from event if set
		if _, ok := event["type"]; ok {
			typeName, ok = event["type"].(string)
			if !ok {
				logp.Err("Type couldn't be overwritten because event type is not string")
			}
			delete(event, "type")
		}

		// Set timestamp from event if set, move it to the top level
		// If not set, timestamp is created
		if _, ok := event["@timestamp"]; ok {
			timestamp, ok = event["@timestamp"].(common.Time)
			if !ok {
				logp.Err("Timestamp couldn't be overwritten because event @timestamp is not common.Time")
			}
			delete(event, "@timestamp")
		}

		eventFieldName := m.Config.Module + "-" + m.Name

		// TODO: Add fields_under_root option for "metrics"?
		event = common.MapStr{
			"type":         typeName,
			eventFieldName: event,
			"metricset":    m.Name,
			"module":       m.Config.Module,
			"@timestamp":   timestamp,
		}

		// Overwrite index in case it is set
		if indexName != "" {
			event["beat"] = common.MapStr{
				"index": indexName,
			}
		}

		newEvents = append(newEvents, event)
	}

	return newEvents, nil
}
