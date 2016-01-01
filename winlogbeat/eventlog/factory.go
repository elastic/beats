package eventlog

import (
	"fmt"
	"sort"
	"strings"
)

// Config is the configuration data used to instantiate a new EventLog.
type Config struct {
	Name          string // Name of the event log or channel.
	RemoteAddress string // Remote computer to connect to. Optional.

	API string // Name of the API to use. Optional.
}

// Producer produces a new event log instance for reading event log records.
type producer func(Config) (EventLog, error)

// Channels lists the available channels (event logs).
type channels func() ([]string, error)

// eventLogInfo is the registration info associate with an event log API.
type eventLogInfo struct {
	apiName  string
	priority int
	producer producer
	channels func() ([]string, error)
}

// eventLogs is a map of priorities to eventLogInfo. The lower numbers have
// higher priorities.
var eventLogs = make(map[int]eventLogInfo)

// Register registers an EventLog API. Only the APIs that are available for the
// runtime OS should be registered. Each API must have a unique priority.
func Register(apiName string, priority int, producer producer, channels channels) {
	info, exists := eventLogs[priority]
	if exists {
		panic(fmt.Sprintf("%s API is already registered with priority %d. "+
			"Cannot register %s", info.apiName, info.priority, apiName))
	}

	eventLogs[priority] = eventLogInfo{
		apiName:  apiName,
		priority: priority,
		producer: producer,
		channels: channels,
	}
}

// New creates and returns a new EventLog instance based on the given config
// and the registered EventLog producers.
func New(config Config) (EventLog, error) {
	if len(eventLogs) == 0 {
		return nil, fmt.Errorf("No event log API is available on this system")
	}

	// A specific API is being requested (usually done for testing).
	if config.API != "" {
		for _, v := range eventLogs {
			debugf("Testing %s", v.apiName)
			if strings.EqualFold(v.apiName, config.API) {
				debugf("Using %s API for event log %s", v.apiName, config.Name)
				e, err := v.producer(config)
				return e, err
			}
		}

		return nil, fmt.Errorf("%s API is not available", config.API)
	}

	// Use the API with the highest priority.
	keys := make([]int, 0, len(eventLogs))
	for key := range eventLogs {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	eventLog := eventLogs[keys[0]]
	debugf("Using highest priority API, %s, for event log %s",
		eventLog.apiName, config.Name)
	e, err := eventLog.producer(config)
	return e, err
}
