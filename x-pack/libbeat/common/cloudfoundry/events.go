// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/cloudfoundry/sonde-go/events"
)

// EventType defines the different event types that can be raised from RPLClient.
type EventType uint

// EventTypes from loggregator documented here: https://github.com/cloudfoundry/loggregator-api
const (
	// EventTypeHttpAccess is a http access event.
	EventTypeHttpAccess EventType = iota
	// EventTypeLog is a log event.
	EventTypeLog
	// EventTypeCounter is a counter event.
	EventTypeCounter
	// EventTypeValueMetric is a value metric event.
	EventTypeValueMetric
	// EventTypeContainerMetric is a container metric event.
	EventTypeContainerMetric
	// EventTypeError is an error event.
	EventTypeError
)

// String returns string representation of the event type.
func (t EventType) String() string {
	switch t {
	case EventTypeHttpAccess:
		return "access"
	case EventTypeLog:
		return "log"
	case EventTypeCounter:
		return "counter"
	case EventTypeValueMetric:
		return "value"
	case EventTypeContainerMetric:
		return "container"
	case EventTypeError:
		return "error"
	default:
		return "unknown"
	}
}

// EventMessageType defines the different log message types.
type EventLogMessageType uint

const (
	// EventLogMessageTypeStdout is a message that was received from stdout.
	EventLogMessageTypeStdout EventLogMessageType = iota + 1
	// EventLogMessageTypeStderr is a message that was received from stderr.
	EventLogMessageTypeStderr
)

// String returns string representation of the event log message type.
func (t EventLogMessageType) String() string {
	switch t {
	case EventLogMessageTypeStdout:
		return "stdout"
	case EventLogMessageTypeStderr:
		return "stderr"
	default:
		return "unknown"
	}
}

// Event is the interface all events implements.
type Event interface {
	fmt.Stringer

	Origin() string
	EventType() EventType
	Timestamp() time.Time
	Deployment() string
	Job() string
	Index() string
	IP() string
	Tags() map[string]string
	ToFields() mapstr.M
}

// EventWithAppID is the interface all events implement that provide an application ID for the event.
type EventWithAppID interface {
	Event

	AppGuid() string
}

type eventBase struct {
	origin     string
	timestamp  time.Time
	deployment string
	job        string
	index      string
	ip         string
	tags       map[string]string
}

type eventAppBase struct {
	eventBase

	appGuid string
}

// EventHttpAccess represents a http access event.
type EventHttpAccess struct {
	eventAppBase

	startTimestamp time.Time
	stopTimestamp  time.Time
	requestID      string
	peerType       string
	method         string
	uri            string
	remoteAddress  string
	userAgent      string
	statusCode     int32
	contentLength  int64
	instanceIndex  int32
	forwarded      []string
}

func (*EventHttpAccess) EventType() EventType        { return EventTypeHttpAccess }
func (e *EventHttpAccess) String() string            { return e.EventType().String() }
func (e *EventHttpAccess) Origin() string            { return e.origin }
func (e *EventHttpAccess) Timestamp() time.Time      { return e.timestamp }
func (e *EventHttpAccess) Deployment() string        { return e.deployment }
func (e *EventHttpAccess) Job() string               { return e.job }
func (e *EventHttpAccess) Index() string             { return e.index }
func (e *EventHttpAccess) IP() string                { return e.ip }
func (e *EventHttpAccess) Tags() map[string]string   { return e.tags }
func (e *EventHttpAccess) AppGuid() string           { return e.appGuid }
func (e *EventHttpAccess) StartTimestamp() time.Time { return e.startTimestamp }
func (e *EventHttpAccess) StopTimestamp() time.Time  { return e.stopTimestamp }
func (e *EventHttpAccess) RequestID() string         { return e.requestID }
func (e *EventHttpAccess) PeerType() string          { return e.peerType }
func (e *EventHttpAccess) Method() string            { return e.method }
func (e *EventHttpAccess) URI() string               { return e.uri }
func (e *EventHttpAccess) RemoteAddress() string     { return e.remoteAddress }
func (e *EventHttpAccess) UserAgent() string         { return e.userAgent }
func (e *EventHttpAccess) StatusCode() int32         { return e.statusCode }
func (e *EventHttpAccess) ContentLength() int64      { return e.contentLength }
func (e *EventHttpAccess) InstanceIndex() int32      { return e.instanceIndex }
func (e *EventHttpAccess) Forwarded() []string       { return e.forwarded }
func (e *EventHttpAccess) ToFields() mapstr.M {
	fields := baseMapWithApp(e)
	fields.DeepUpdate(mapstr.M{
		"http": mapstr.M{
			"response": mapstr.M{
				"status_code": e.StatusCode(),
				"method":      e.Method(),
				"bytes":       e.ContentLength(),
			},
		},
		"user_agent": mapstr.M{
			"original": e.UserAgent(),
		},
		"url": urlMap(e.URI()),
	})
	return fields
}

// EventLog represents a log message event.
type EventLog struct {
	eventAppBase

	message     string
	messageType EventLogMessageType
	sourceType  string
	sourceID    string
}

func (*EventLog) EventType() EventType               { return EventTypeLog }
func (e *EventLog) String() string                   { return e.EventType().String() }
func (e *EventLog) Origin() string                   { return e.origin }
func (e *EventLog) Timestamp() time.Time             { return e.timestamp }
func (e *EventLog) Deployment() string               { return e.deployment }
func (e *EventLog) Job() string                      { return e.job }
func (e *EventLog) Index() string                    { return e.index }
func (e *EventLog) IP() string                       { return e.ip }
func (e *EventLog) Tags() map[string]string          { return e.tags }
func (e *EventLog) AppGuid() string                  { return e.appGuid }
func (e *EventLog) Message() string                  { return e.message }
func (e *EventLog) MessageType() EventLogMessageType { return e.messageType }
func (e *EventLog) SourceType() string               { return e.sourceType }
func (e *EventLog) SourceID() string                 { return e.sourceID }
func (e *EventLog) ToFields() mapstr.M {
	fields := baseMapWithApp(e)
	fields.DeepUpdate(mapstr.M{
		"cloudfoundry": mapstr.M{
			e.String(): mapstr.M{
				"source": mapstr.M{
					"instance": e.SourceID(),
					"type":     e.SourceType(),
				},
			},
		},
		"message": e.Message(),
		"stream":  e.MessageType().String(),
	})
	return fields
}

// EventCounter represents a counter event.
type EventCounter struct {
	eventBase

	name  string
	delta uint64
	total uint64
}

func (*EventCounter) EventType() EventType      { return EventTypeCounter }
func (e *EventCounter) String() string          { return e.EventType().String() }
func (e *EventCounter) Origin() string          { return e.origin }
func (e *EventCounter) Timestamp() time.Time    { return e.timestamp }
func (e *EventCounter) Deployment() string      { return e.deployment }
func (e *EventCounter) Job() string             { return e.job }
func (e *EventCounter) Index() string           { return e.index }
func (e *EventCounter) IP() string              { return e.ip }
func (e *EventCounter) Tags() map[string]string { return e.tags }
func (e *EventCounter) Name() string            { return e.name }
func (e *EventCounter) Delta() uint64           { return e.delta }
func (e *EventCounter) Total() uint64           { return e.total }
func (e *EventCounter) ToFields() mapstr.M {
	fields := baseMap(e)
	fields.DeepUpdate(mapstr.M{
		"cloudfoundry": mapstr.M{
			e.String(): mapstr.M{
				"name":  e.Name(),
				"delta": e.Delta(),
				"total": e.Total(),
			},
		},
	})
	return fields
}

// EventValueMetric represents a value metric event.
type EventValueMetric struct {
	eventBase

	name  string
	value float64
	unit  string
}

func (*EventValueMetric) EventType() EventType      { return EventTypeValueMetric }
func (e *EventValueMetric) String() string          { return e.EventType().String() }
func (e *EventValueMetric) Origin() string          { return e.origin }
func (e *EventValueMetric) Timestamp() time.Time    { return e.timestamp }
func (e *EventValueMetric) Deployment() string      { return e.deployment }
func (e *EventValueMetric) Job() string             { return e.job }
func (e *EventValueMetric) Index() string           { return e.index }
func (e *EventValueMetric) IP() string              { return e.ip }
func (e *EventValueMetric) Tags() map[string]string { return e.tags }
func (e *EventValueMetric) Name() string            { return e.name }
func (e *EventValueMetric) Value() float64          { return e.value }
func (e *EventValueMetric) Unit() string            { return e.unit }
func (e *EventValueMetric) ToFields() mapstr.M {
	fields := baseMap(e)
	fields.DeepUpdate(mapstr.M{
		"cloudfoundry": mapstr.M{
			e.String(): mapstr.M{
				"name":  e.Name(),
				"unit":  e.Unit(),
				"value": e.Value(),
			},
		},
	})
	return fields
}

// EventContainerMetric represents a container metric event.
type EventContainerMetric struct {
	eventAppBase

	instanceIndex    int32
	cpuPercentage    float64
	memoryBytes      uint64
	diskBytes        uint64
	memoryBytesQuota uint64
	diskBytesQuota   uint64
}

func (*EventContainerMetric) EventType() EventType       { return EventTypeContainerMetric }
func (e *EventContainerMetric) String() string           { return e.EventType().String() }
func (e *EventContainerMetric) Origin() string           { return e.origin }
func (e *EventContainerMetric) Timestamp() time.Time     { return e.timestamp }
func (e *EventContainerMetric) Deployment() string       { return e.deployment }
func (e *EventContainerMetric) Job() string              { return e.job }
func (e *EventContainerMetric) Index() string            { return e.index }
func (e *EventContainerMetric) IP() string               { return e.ip }
func (e *EventContainerMetric) Tags() map[string]string  { return e.tags }
func (e *EventContainerMetric) AppGuid() string          { return e.appGuid }
func (e *EventContainerMetric) InstanceIndex() int32     { return e.instanceIndex }
func (e *EventContainerMetric) CPUPercentage() float64   { return e.cpuPercentage }
func (e *EventContainerMetric) MemoryBytes() uint64      { return e.memoryBytes }
func (e *EventContainerMetric) DiskBytes() uint64        { return e.diskBytes }
func (e *EventContainerMetric) MemoryBytesQuota() uint64 { return e.memoryBytesQuota }
func (e *EventContainerMetric) DiskBytesQuota() uint64   { return e.diskBytesQuota }
func (e *EventContainerMetric) ToFields() mapstr.M {
	fields := baseMapWithApp(e)
	fields.DeepUpdate(mapstr.M{
		"cloudfoundry": mapstr.M{
			e.String(): mapstr.M{
				"instance_index":     e.InstanceIndex(),
				"cpu.pct":            e.CPUPercentage(),
				"memory.bytes":       e.MemoryBytes(),
				"memory.quota.bytes": e.MemoryBytesQuota(),
				"disk.bytes":         e.DiskBytes(),
				"disk.quota.bytes":   e.DiskBytesQuota(),
			},
		},
	})
	return fields
}

// EventError represents an error event.
type EventError struct {
	eventBase

	message string
	code    int32
	source  string
}

func (*EventError) EventType() EventType      { return EventTypeError }
func (e *EventError) String() string          { return e.EventType().String() }
func (e *EventError) Origin() string          { return e.origin }
func (e *EventError) Timestamp() time.Time    { return e.timestamp }
func (e *EventError) Deployment() string      { return e.deployment }
func (e *EventError) Job() string             { return e.job }
func (e *EventError) Index() string           { return e.index }
func (e *EventError) IP() string              { return e.ip }
func (e *EventError) Tags() map[string]string { return e.tags }
func (e *EventError) Message() string         { return e.message }
func (e *EventError) Code() int32             { return e.code }
func (e *EventError) Source() string          { return e.source }
func (e *EventError) ToFields() mapstr.M {
	fields := baseMap(e)
	fields.DeepUpdate(mapstr.M{
		"cloudfoundry": mapstr.M{
			e.String(): mapstr.M{
				"source": e.Source(),
			},
		},
		"message": e.Message(),
		"code":    e.Code(),
	})
	return fields
}

func newEventBase(env *events.Envelope) eventBase {
	return eventBase{
		origin:     *env.Origin,
		timestamp:  time.Unix(0, *env.Timestamp),
		deployment: *env.Deployment,
		job:        *env.Job,
		index:      *env.Index,
		ip:         *env.Ip,
		tags:       env.Tags,
	}
}

func newEventHttpAccess(env *events.Envelope) *EventHttpAccess {
	msg := env.GetHttpStartStop()
	e := EventHttpAccess{
		eventAppBase: eventAppBase{
			eventBase: newEventBase(env),
			appGuid:   formatUUID(msg.ApplicationId),
		},
		startTimestamp: time.Unix(0, *msg.StartTimestamp),
		stopTimestamp:  time.Unix(0, *msg.StopTimestamp),
		requestID:      formatUUID(msg.RequestId),
		peerType:       strings.ToLower(msg.PeerType.String()),
		method:         msg.Method.String(),
		uri:            *msg.Uri,
		remoteAddress:  *msg.RemoteAddress,
		userAgent:      *msg.UserAgent,
		statusCode:     *msg.StatusCode,
		contentLength:  *msg.ContentLength,
		forwarded:      msg.Forwarded,
	}
	if msg.InstanceIndex != nil {
		e.instanceIndex = *msg.InstanceIndex
	}
	return &e
}

func newEventLog(env *events.Envelope) *EventLog {
	msg := env.GetLogMessage()
	return &EventLog{
		eventAppBase: eventAppBase{
			eventBase: newEventBase(env),
			appGuid:   *msg.AppId,
		},
		message:     string(msg.Message),
		messageType: EventLogMessageType(*msg.MessageType),
		sourceType:  *msg.SourceType,
		sourceID:    *msg.SourceInstance,
	}
}

func newEventCounter(env *events.Envelope) *EventCounter {
	msg := env.GetCounterEvent()
	return &EventCounter{
		eventBase: newEventBase(env),
		name:      *msg.Name,
		delta:     *msg.Delta,
		total:     *msg.Total,
	}
}

func newEventValueMetric(env *events.Envelope) *EventValueMetric {
	msg := env.GetValueMetric()
	return &EventValueMetric{
		eventBase: newEventBase(env),
		name:      *msg.Name,
		value:     *msg.Value,
		unit:      *msg.Unit,
	}
}

func newEventContainerMetric(env *events.Envelope) *EventContainerMetric {
	msg := env.GetContainerMetric()
	return &EventContainerMetric{
		eventAppBase: eventAppBase{
			eventBase: newEventBase(env),
			appGuid:   *msg.ApplicationId,
		},
		instanceIndex:    *msg.InstanceIndex,
		cpuPercentage:    *msg.CpuPercentage,
		memoryBytes:      *msg.MemoryBytes,
		diskBytes:        *msg.DiskBytes,
		memoryBytesQuota: *msg.MemoryBytesQuota,
		diskBytesQuota:   *msg.DiskBytesQuota,
	}
}

func newEventError(env *events.Envelope) *EventError {
	msg := env.GetError()
	return &EventError{
		eventBase: newEventBase(env),
		message:   *msg.Message,
		code:      *msg.Code,
		source:    *msg.Source,
	}
}

func EnvelopeToEvent(env *events.Envelope) Event {
	switch *env.EventType {
	case events.Envelope_HttpStartStop:
		return newEventHttpAccess(env)
	case events.Envelope_LogMessage:
		return newEventLog(env)
	case events.Envelope_CounterEvent:
		return newEventCounter(env)
	case events.Envelope_ValueMetric:
		return newEventValueMetric(env)
	case events.Envelope_ContainerMetric:
		return newEventContainerMetric(env)
	case events.Envelope_Error:
		return newEventError(env)
	}
	return nil
}

func envelopMap(evt Event) mapstr.M {
	return mapstr.M{
		"origin":     evt.Origin(),
		"deployment": evt.Deployment(),
		"ip":         evt.IP(),
		"job":        evt.Job(),
		"index":      evt.Index(),
	}
}

func baseMap(evt Event) mapstr.M {
	tags, meta := tagsToMeta(evt.Tags())
	cf := mapstr.M{
		"type":     evt.String(),
		"envelope": envelopMap(evt),
	}
	if len(tags) > 0 {
		cf["tags"] = tags
	}
	result := mapstr.M{
		"cloudfoundry": cf,
	}
	if len(meta) > 0 {
		result.DeepUpdate(meta)
	}
	return result
}

func tagsToMeta(eventTags map[string]string) (tags mapstr.M, meta mapstr.M) {
	tags = mapstr.M{}
	meta = mapstr.M{}
	for name, value := range eventTags {
		switch name {
		case "app_id":
			meta.Put("cloudfoundry.app.id", value)
		case "app_name":
			meta.Put("cloudfoundry.app.name", value)
		case "space_id":
			meta.Put("cloudfoundry.space.id", value)
		case "space_name":
			meta.Put("cloudfoundry.space.name", value)
		case "organization_id":
			meta.Put("cloudfoundry.org.id", value)
		case "organization_name":
			meta.Put("cloudfoundry.org.name", value)
		default:
			tags[common.DeDot(name)] = value
		}
	}
	return tags, meta
}

func baseMapWithApp(evt EventWithAppID) mapstr.M {
	base := baseMap(evt)
	appID := evt.AppGuid()
	if appID != "" {
		base.Put("cloudfoundry.app.id", appID)
	}
	return base
}

func urlMap(uri string) mapstr.M {
	u, err := url.Parse(uri)
	if err != nil {
		return mapstr.M{
			"original": uri,
		}
	}
	return mapstr.M{
		"original": uri,
		"scheme":   u.Scheme,
		"port":     u.Port(),
		"path":     u.Path,
		"domain":   u.Hostname(),
	}
}

func formatUUID(uuid *events.UUID) string {
	if uuid == nil {
		return ""
	}
	var uuidBytes [16]byte
	binary.LittleEndian.PutUint64(uuidBytes[:8], uuid.GetLow())
	binary.LittleEndian.PutUint64(uuidBytes[8:], uuid.GetHigh())
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:])
}
