package module

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/metricbeat/mb"
)

// EventBuilder is used for building MetricSet events. MetricSets generate a
// data in the form of a common.MapStr. This builder transforms that data into
// a complete event and applies any Module-level filtering.
type EventBuilder struct {
	ModuleName    string
	MetricSetName string
	Host          string
	StartTime     time.Time
	FetchDuration time.Duration
	Event         common.MapStr
	fetchErr      error
}

// Build builds an event from MetricSet data and applies the Module-level
// filters.
func (b EventBuilder) Build() (beat.Event, error) {
	// event may be nil when there was an error fetching.
	fields := b.Event
	if fields == nil {
		fields = common.MapStr{}
	}

	// Get and remove meta fields from the event created by the MetricSet.
	timestamp := time.Time(getTimestamp(fields, common.Time(b.StartTime)))

	metricsetData := common.MapStr{
		"module": b.ModuleName,
		"name":   b.MetricSetName,
	}
	// Adds host name to event.
	if b.Host != "" {
		metricsetData["host"] = b.Host
	}
	if b.FetchDuration != 0 {
		metricsetData["rtt"] = b.FetchDuration.Nanoseconds() / int64(time.Microsecond)
	}

	namespace := b.MetricSetName
	if n, ok := fields["_namespace"]; ok {
		delete(fields, "_namespace")
		if ns, ok := n.(string); ok {
			namespace = ns
		}

		metricsetData["namespace"] = namespace
	}

	// Checks if additional meta information is provided by the MetricSet under the key ModuleData
	// This is based on the convention that each MetricSet can provide module data under the key ModuleData
	moduleData, moudleDataExists := fields[mb.ModuleDataKey]
	if moudleDataExists {
		delete(fields, mb.ModuleDataKey)
	}

	moduleEvent := common.MapStr{}
	moduleEvent.Put(namespace, fields)

	// In case meta data exists, it is added on the module level
	// This is mostly used for shared fields across multiple metricsets in one module
	if moudleDataExists {
		if data, ok := moduleData.(common.MapStr); ok {
			moduleEvent.DeepUpdate(data)
		}
	}

	event := beat.Event{
		Timestamp: time.Time(timestamp),
		Fields: common.MapStr{
			b.ModuleName: moduleEvent,
			"metricset":  metricsetData,
		},
	}

	// Adds error to event in case error happened
	if b.fetchErr != nil {
		event.Fields["error"] = common.MapStr{
			"message": b.fetchErr.Error(),
		}
	}

	return event, nil
}

// getTimestamp gets the @timestamp field from the event, removes the key from
// the event, and returns the value. If the key is not present or not the proper
// type then the provided timestamp value is returned instead.
func getTimestamp(event common.MapStr, timestamp common.Time) common.Time {
	if ts, found := event["@timestamp"]; found {
		delete(event, "@timestamp")

		switch v := ts.(type) {
		case common.Time:
			timestamp = v
		case time.Time:
			timestamp = common.Time(v)
		default:
			logp.Err("Ignoring @timestamp value because its type (%T) is not "+
				"common.Time or time.Time", v)
		}
	}
	return timestamp
}

// createEvent creates a new event from the fetched MetricSet data.
func createEvent(
	msw *metricSetWrapper,
	event common.MapStr,
	fetchErr error,
	start time.Time,
	elapsed time.Duration,
) (beat.Event, error) {
	return EventBuilder{
		ModuleName:    msw.Module().Name(),
		MetricSetName: msw.Name(),
		Host:          msw.Host(),
		StartTime:     start,
		FetchDuration: elapsed,
		Event:         event,
		fetchErr:      fetchErr,
	}.Build()
}
