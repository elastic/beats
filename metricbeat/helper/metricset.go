package helper

import (
	"expvar"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// expvar variables
var (
	fetchedEvents        = expvar.NewMap("fetchedEvents")
	openMetricSetFetches = expvar.NewMap("openMetricSetFetches")
)

const (
	// To prevent "too many open files" issue the number of concurrent fetchers per
	// metricset instance is limited to 32. Concurrent fetching happens if the timeout
	// is set bigger then period or if a fetch method does not timeout properly.
	maxConcurrentFetchers = 32
)

// Metric specific data
// This must be defined by each metric
type MetricSet struct {
	Name        string
	MetricSeter MetricSeter
	// Inherits config from module
	Config ModuleConfig
	Module *Module

	fetchCounter uint32
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

		go func(h string) {

			fetchCounter := m.incrementFetcher()
			defer m.decrementFetcher()

			var event common.MapStr
			var err error

			starttime := time.Now()

			// Check if max number of concurrent fetchers is reached
			if fetchCounter > maxConcurrentFetchers {
				err = fmt.Errorf("Too many concurrent fetchers started for metricset %s", m.Name)
				logp.Err("Too many concurrent fetchers started for metricset %s", m.Name)
			} else {
				// Fetch method must make sure to return error after Timeout reached
				event, err = m.MetricSeter.Fetch(m, h)
			}
			elapsed := time.Since(starttime)

			// expvar stats
			baseName := m.Module.name + "-" + m.Name
			if err != nil {
				fetchedEvents.Add(baseName+"-failed", 1)
			} else {
				fetchedEvents.Add(baseName+"-success", 1)
			}

			event = m.createEvent(event, h, elapsed, err)

			m.Module.Publish <- event

		}(host)
	}
	return nil
}

func (m *MetricSet) createEvent(event common.MapStr, host string, rtt time.Duration, eventErr error) common.MapStr {

	// Most of the time, event is nil in case of error (not required)
	if event == nil {
		event = common.MapStr{}
	}

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

	// Makes sure filters are set and applies them
	if m.Module.filters != nil {
		event = m.Module.filters.Filter(event)
	}

	event = common.MapStr{
		"type":                  typeName,
		eventFieldName:          event,
		"metricset":             m.Name,
		"module":                m.Module.name,
		"rtt":                   rtt.Nanoseconds() / int64(time.Microsecond),
		"@timestamp":            timestamp,
		common.EventMetadataKey: m.Config.EventMetadata,
	}

	// Overwrite index in case it is set
	if indexName != "" {
		event["beat"] = common.MapStr{
			"index": indexName,
		}
	}

	// Adds host name to event. In case credentials are passed through hostname, these are contained in this string
	if host != "" {
		event["metricset-host"] = host
	}

	// Adds error to event in case error happened
	if eventErr != nil {
		event["error"] = eventErr.Error()
	}

	return event
}

// incrementFetcher increments the number of open fetcher
func (m *MetricSet) incrementFetcher() uint32 {
	openMetricSetFetches.Add(m.Module.name+"-"+m.Name, 1)
	return atomic.AddUint32(&m.fetchCounter, 1)
}

// decrementFetcher decrements the number of open fetchers
func (m *MetricSet) decrementFetcher() uint32 {
	openMetricSetFetches.Add(m.Module.name+"-"+m.Name, -1)
	// Decrements value by 1
	return atomic.AddUint32(&m.fetchCounter, ^uint32(0))
}
