package helper

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"gopkg.in/yaml.v2"
	"sync"
	"time"
)

// Base metric configuration
type MetricSetConfig struct {
	Period string
}

// Metric specific data
// This must be defined by each metric
type MetricSet struct {
	Name    string
	Enabled bool

	// Generic Config existing in all metrics
	BaseConfig MetricSetConfig

	// Raw metric specific config
	// This is provided to convert it into Config later
	RawConfig interface{}

	// Metric specific config
	Config interface{}

	MetricSeter MetricSeter
	Module      *Module

	// Control channel
	done chan struct{}
}

// Interface for each metric
type MetricSeter interface {
	// Setup needed for all upcoming fetches
	// Typically config is loaded here
	Setup() error

	// Method to periodically fetch new events
	Fetch() ([]common.MapStr, error)

	// Cleanup when stopping metricset
	Cleanup() error
}

// Creates a new MetricSet
func NewMetricSet(name string, metricset MetricSeter, module *Module) *MetricSet {
	return &MetricSet{
		Name:        name,
		MetricSeter: metricset,
		Module:      module,
		Enabled:     false,
		done:        make(chan struct{}),
	}
}

func (m *MetricSet) LoadConfig(config interface{}) {

	bytes, err := yaml.Marshal(m.RawConfig)

	if err != nil {
		logp.Err("Load metric config error: %v", err)
	}
	yaml.Unmarshal(bytes, config)
}

// Registers metric with module
func (m *MetricSet) Register() {
	m.Module.AddMetric(m)
}

// RunMetric runs the given metric
func (m *MetricSet) Start(b *beat.Beat, wg sync.WaitGroup) {

	// Catches metric in case of panic. Keeps other metricsets running
	defer func() {
		if r := recover(); r != nil {
			logp.Err("Metric %s paniced and stopped running. Reason: %+v", m.Name, r)
		}
		wg.Done()
	}()

	// Only starts metricset if enabled
	if !m.Enabled {
		logp.Debug("helper", "Not starting metric %s as not enabled.", m.Name)
		return
	}

	// Setup
	err := m.MetricSeter.Setup()
	if err != nil {
		logp.Err("Error happening during metricseter setup: %s", err)
	}
	period, err := time.ParseDuration(m.BaseConfig.Period)

	if err != nil {
		logp.Info("Error in parsing period of metric %s: %v", m.Name, err)
	}

	// If no period set, set default
	if period == 0 {
		logp.Info("Setting default period for metric %s as not set.", m.Name)
		period = 1 * time.Second
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	logp.Info("Start metric %s with period %v", m.Name, period)

	for {
		select {
		case <-m.done:
			logp.Info("Stopping metricset: %s", m.Name)
			return
		case <-ticker.C:
		}

		events, err := m.MetricSeter.Fetch()
		if err != nil {
			logp.Err("Fetching events in MetricSet %s returned error: %s", m.Name, err)
			continue
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
				indexName = event["index"].(string)
				delete(event, "index")
			}

			// Set type from event if set
			if _, ok := event["type"]; ok {
				typeName = event["type"].(string)
				delete(event, "type")
			}

			// Set timestamp from event if set, move it to the top level
			// If not set, timestamp is created
			if _, ok := event["@timestamp"]; ok {
				timestamp = event["@timestamp"].(common.Time)
				delete(event, "@timestamp")
			}

			eventFieldName := m.Module.Name + "-" + m.Name
			// TODO: Add fields_under_root option for "metrics"?
			event = common.MapStr{
				"type":         typeName,
				eventFieldName: event,
				"metricset":    m.Name,
				"module":       m.Module.Name,
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

		// Async publishing of event
		b.Events.PublishEvents(newEvents)
	}
}

// Stop stops the metricset
func (m *MetricSet) Stop() {
	logp.Info("Stopping metricset: %s", m.Name)
	close(m.done)

	err := m.MetricSeter.Cleanup()
	if err != nil {
		logp.Err("Error cleaning up metricset %s: %s", m.Name, err)
	}
}
