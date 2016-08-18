package beater

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	defaultType = "metricsets"
)

// eventBuilder is used for building MetricSet events. MetricSets generate a
// data in the form of a common.MapStr. This builder transforms that data into
// a complete event and applies any Module-level filtering.
type eventBuilder struct {
	moduleName    string
	metricSetName string
	host          string
	startTime     time.Time
	fetchDuration time.Duration
	event         common.MapStr
	fetchErr      error
	filters       *processors.Processors
	metadata      common.EventMetadata
}

// build builds an event from MetricSet data and applies the Module-level
// filters.
func (b eventBuilder) build() (common.MapStr, error) {
	// event may be nil when there was an error fetching.
	event := b.event
	if event == nil {
		event = common.MapStr{} // TODO (akroh): do we want to send an empty event field?
	}

	// Get and remove meta fields from the event created by the MetricSet.
	indexName := getIndex(event, "")
	typeName := getType(event, defaultType)
	timestamp := getTimestamp(event, common.Time(b.startTime))

	// Apply filters.
	if b.filters != nil {
		if event = b.filters.Run(event); event == nil {
			return nil, nil
		}
	}

	event = common.MapStr{
		"@timestamp": timestamp,
		"type":       typeName,

		common.EventMetadataKey: b.metadata,
		b.moduleName: common.MapStr{
			b.metricSetName: event,
		},
		"metricset": common.MapStr{
			"module": b.moduleName,
			"name":   b.metricSetName,
			"rtt":    b.fetchDuration.Nanoseconds() / int64(time.Microsecond),
		},
	}

	// Overwrite default index if set.
	if indexName != "" {
		event["beat"] = common.MapStr{
			"index": indexName,
		}
	}

	// Adds host name to event. In case credentials are passed through
	// hostname, these are contained in this string.
	if b.host != "" {
		// TODO (akroh): allow metricset to specify this value so that
		// a proper URL can be specified and passwords be redacted.
		event["metricset"].(common.MapStr)["host"] = b.host
	}

	// Adds error to event in case error happened
	if b.fetchErr != nil {
		event["error"] = b.fetchErr.Error()
	}

	return event, nil
}

func getIndex(event common.MapStr, indexName string) string {
	// Set index from event if set

	if _, ok := event["index"]; ok {
		indexName, ok = event["index"].(string)
		if !ok {
			logp.Err("Index couldn't be overwritten because event index is not string")
		}
		delete(event, "index")
	}
	return indexName
}

func getType(event common.MapStr, typeName string) string {

	// Set type from event if set
	if _, ok := event["type"]; ok {
		typeName, ok = event["type"].(string)
		if !ok {
			logp.Err("Type couldn't be overwritten because event type is not string")
		}
		delete(event, "type")
	}

	return typeName
}

func getTimestamp(event common.MapStr, timestamp common.Time) common.Time {

	// Set timestamp from event if set, move it to the top level
	// If not set, timestamp is created
	if _, ok := event["@timestamp"]; ok {
		timestamp, ok = event["@timestamp"].(common.Time)
		if !ok {
			logp.Err("Timestamp couldn't be overwritten because event @timestamp is not common.Time")
		}
		delete(event, "@timestamp")
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
) (common.MapStr, error) {
	return eventBuilder{
		moduleName:    msw.Module().Name(),
		metricSetName: msw.Name(),
		host:          msw.Host(),
		startTime:     start,
		fetchDuration: elapsed,
		event:         event,
		fetchErr:      fetchErr,
		filters:       msw.module.filters,
		metadata:      msw.module.Config().EventMetadata,
	}.build()
}
