// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cloudfoundry

import (
	"testing"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestEventTypeHttpAccess(t *testing.T) {
	eventType := events.Envelope_HttpStartStop
	startTimestamp := int64(1587469726082)
	stopTimestamp := int64(1587469875895)
	peerType := events.PeerType_Client
	method := events.Method_GET
	uri := "https://uri.full-domain.com:8443/subpath"
	remoteAddress := "remote_address"
	userAgent := "user_agent"
	statusCode := int32(200)
	contentLength := int64(128)
	appID := makeUUID()
	instanceIdx := int32(1)
	instanceID := "instance_id"
	forwarded := []string{"forwarded"}
	cfEvt := makeEnvelope(&eventType)
	cfEvt.HttpStartStop = &events.HttpStartStop{
		StartTimestamp: &startTimestamp,
		StopTimestamp:  &stopTimestamp,
		RequestId:      makeUUID(),
		PeerType:       &peerType,
		Method:         &method,
		Uri:            &uri,
		RemoteAddress:  &remoteAddress,
		UserAgent:      &userAgent,
		StatusCode:     &statusCode,
		ContentLength:  &contentLength,
		ApplicationId:  appID,
		InstanceIndex:  &instanceIdx,
		InstanceId:     &instanceID,
		Forwarded:      forwarded,
	}
	evt := newEventHttpAccess(cfEvt)

	assert.Equal(t, EventTypeHttpAccess, evt.EventType())
	assert.Equal(t, "access", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, map[string]string{"tag": "value"}, evt.Tags())
	assert.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", evt.AppGuid())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.StartTimestamp())
	assert.Equal(t, time.Unix(0, 1587469875895), evt.StopTimestamp())
	assert.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", evt.RequestID())
	assert.Equal(t, "client", evt.PeerType())
	assert.Equal(t, "GET", evt.Method())
	assert.Equal(t, "https://uri.full-domain.com:8443/subpath", evt.URI())
	assert.Equal(t, "remote_address", evt.RemoteAddress())
	assert.Equal(t, "user_agent", evt.UserAgent())
	assert.Equal(t, int32(200), evt.StatusCode())
	assert.Equal(t, int64(128), evt.ContentLength())
	assert.Equal(t, int32(1), evt.InstanceIndex())
	assert.Equal(t, []string{"forwarded"}, evt.Forwarded())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "access",
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"app": mapstr.M{
				"id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			},
			"tags": mapstr.M{
				"tag": "value",
			},
		},
		"http": mapstr.M{
			"response": mapstr.M{
				"status_code": int32(200),
				"method":      "GET",
				"bytes":       int64(128),
			},
		},
		"user_agent": mapstr.M{
			"original": "user_agent",
		},
		"url": mapstr.M{
			"original": "https://uri.full-domain.com:8443/subpath",
			"scheme":   "https",
			"port":     "8443",
			"path":     "/subpath",
			"domain":   "uri.full-domain.com",
		},
	}, evt.ToFields())
}

func TestEventTypeLog(t *testing.T) {
	eventType := events.Envelope_LogMessage
	message := "log message"
	messageType := events.LogMessage_OUT
	timestamp := int64(1587469726082)
	appID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	sourceType := "source_type"
	sourceInstance := "source_instance"
	cfEvt := makeEnvelope(&eventType)
	cfEvt.LogMessage = &events.LogMessage{
		Message:        []byte(message),
		MessageType:    &messageType,
		Timestamp:      &timestamp,
		AppId:          &appID,
		SourceType:     &sourceType,
		SourceInstance: &sourceInstance,
	}
	evt := newEventLog(cfEvt)

	assert.Equal(t, EventTypeLog, evt.EventType())
	assert.Equal(t, "log", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, map[string]string{"tag": "value"}, evt.Tags())
	assert.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", evt.AppGuid())
	assert.Equal(t, "log message", evt.Message())
	assert.Equal(t, EventLogMessageTypeStdout, evt.MessageType())
	assert.Equal(t, "source_type", evt.SourceType())
	assert.Equal(t, "source_instance", evt.SourceID())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "log",
			"log": mapstr.M{
				"source": mapstr.M{
					"instance": evt.SourceID(),
					"type":     evt.SourceType(),
				},
			},
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"app": mapstr.M{
				"id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			},
			"tags": mapstr.M{
				"tag": "value",
			},
		},
		"message": "log message",
		"stream":  "stdout",
	}, evt.ToFields())
}

func TestEventCounter(t *testing.T) {
	eventType := events.Envelope_CounterEvent
	name := "name"
	delta := uint64(10)
	total := uint64(999)
	cfEvt := makeEnvelope(&eventType)
	cfEvt.CounterEvent = &events.CounterEvent{
		Name:  &name,
		Delta: &delta,
		Total: &total,
	}
	evt := newEventCounter(cfEvt)

	assert.Equal(t, EventTypeCounter, evt.EventType())
	assert.Equal(t, "counter", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, map[string]string{"tag": "value"}, evt.Tags())
	assert.Equal(t, "name", evt.Name())
	assert.Equal(t, uint64(10), evt.Delta())
	assert.Equal(t, uint64(999), evt.Total())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "counter",
			"counter": mapstr.M{
				"name":  "name",
				"delta": uint64(10),
				"total": uint64(999),
			},
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"tags": mapstr.M{
				"tag": "value",
			},
		},
	}, evt.ToFields())
}

func TestEventValueMetric(t *testing.T) {
	eventType := events.Envelope_ValueMetric
	name := "name"
	value := 10.1
	unit := "unit"
	cfEvt := makeEnvelope(&eventType)
	cfEvt.ValueMetric = &events.ValueMetric{
		Name:  &name,
		Value: &value,
		Unit:  &unit,
	}
	evt := newEventValueMetric(cfEvt)

	assert.Equal(t, EventTypeValueMetric, evt.EventType())
	assert.Equal(t, "value", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, map[string]string{"tag": "value"}, evt.Tags())
	assert.Equal(t, "name", evt.Name())
	assert.Equal(t, 10.1, evt.Value())
	assert.Equal(t, "unit", evt.Unit())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "value",
			"value": mapstr.M{
				"name":  "name",
				"value": 10.1,
				"unit":  "unit",
			},
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"tags": mapstr.M{
				"tag": "value",
			},
		},
	}, evt.ToFields())
}

func TestEventContainerMetric(t *testing.T) {
	eventType := events.Envelope_ContainerMetric
	appID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	instanceIdx := int32(1)
	cpuPercentage := 0.2
	memoryBytes := uint64(1024)
	diskBytes := uint64(2048)
	memoryBytesQuota := uint64(2048)
	diskBytesQuota := uint64(4096)
	cfEvt := makeEnvelope(&eventType)
	cfEvt.ContainerMetric = &events.ContainerMetric{
		ApplicationId:    &appID,
		InstanceIndex:    &instanceIdx,
		CpuPercentage:    &cpuPercentage,
		MemoryBytes:      &memoryBytes,
		DiskBytes:        &diskBytes,
		MemoryBytesQuota: &memoryBytesQuota,
		DiskBytesQuota:   &diskBytesQuota,
	}
	evt := newEventContainerMetric(cfEvt)

	assert.Equal(t, EventTypeContainerMetric, evt.EventType())
	assert.Equal(t, "container", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, map[string]string{"tag": "value"}, evt.Tags())
	assert.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", evt.AppGuid())
	assert.Equal(t, int32(1), evt.InstanceIndex())
	assert.Equal(t, 0.2, evt.CPUPercentage())
	assert.Equal(t, uint64(1024), evt.MemoryBytes())
	assert.Equal(t, uint64(2048), evt.DiskBytes())
	assert.Equal(t, uint64(2048), evt.MemoryBytesQuota())
	assert.Equal(t, uint64(4096), evt.DiskBytesQuota())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "container",
			"container": mapstr.M{
				"instance_index":     int32(1),
				"cpu.pct":            0.2,
				"memory.bytes":       uint64(1024),
				"memory.quota.bytes": uint64(2048),
				"disk.bytes":         uint64(2048),
				"disk.quota.bytes":   uint64(4096),
			},
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"app": mapstr.M{
				"id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			},
			"tags": mapstr.M{
				"tag": "value",
			},
		},
	}, evt.ToFields())
}

func TestEventError(t *testing.T) {
	eventType := events.Envelope_Error
	source := "source"
	code := int32(100)
	message := "message"
	cfEvt := makeEnvelope(&eventType)
	cfEvt.Error = &events.Error{
		Source:  &source,
		Code:    &code,
		Message: &message,
	}
	evt := newEventError(cfEvt)

	assert.Equal(t, EventTypeError, evt.EventType())
	assert.Equal(t, "error", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, map[string]string{"tag": "value"}, evt.Tags())
	assert.Equal(t, "message", evt.Message())
	assert.Equal(t, int32(100), evt.Code())
	assert.Equal(t, "source", evt.Source())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "error",
			"error": mapstr.M{
				"source": "source",
			},
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"tags": mapstr.M{
				"tag": "value",
			},
		},
		"message": "message",
		"code":    int32(100),
	}, evt.ToFields())
}

func TestEventTagsWithMetadata(t *testing.T) {
	eventType := events.Envelope_LogMessage
	message := "log message"
	messageType := events.LogMessage_OUT
	timestamp := int64(1587469726082)
	appID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	sourceType := "source_type"
	sourceInstance := "source_instance"
	cfEvt := makeEnvelope(&eventType)
	tags := map[string]string{
		"app_id":            appID,
		"app_name":          "some-app",
		"space_id":          "e1114e92-155c-11eb-ada9-27b81025a657",
		"space_name":        "some-space",
		"organization_id":   "baeef1ba-155c-11eb-a1af-8f14964c35d2",
		"organization_name": "some-org",
		"custom_tag":        "foo",
	}
	cfEvt.Tags = tags
	cfEvt.LogMessage = &events.LogMessage{
		Message:        []byte(message),
		MessageType:    &messageType,
		Timestamp:      &timestamp,
		AppId:          &appID,
		SourceType:     &sourceType,
		SourceInstance: &sourceInstance,
	}
	evt := newEventLog(cfEvt)

	assert.Equal(t, EventTypeLog, evt.EventType())
	assert.Equal(t, "log", evt.String())
	assert.Equal(t, "origin", evt.Origin())
	assert.Equal(t, time.Unix(0, 1587469726082), evt.Timestamp())
	assert.Equal(t, "deployment", evt.Deployment())
	assert.Equal(t, "job", evt.Job())
	assert.Equal(t, "index", evt.Index())
	assert.Equal(t, "ip", evt.IP())
	assert.Equal(t, tags, evt.Tags())
	assert.Equal(t, "f47ac10b-58cc-4372-a567-0e02b2c3d479", evt.AppGuid())
	assert.Equal(t, "log message", evt.Message())
	assert.Equal(t, EventLogMessageTypeStdout, evt.MessageType())
	assert.Equal(t, "source_type", evt.SourceType())
	assert.Equal(t, "source_instance", evt.SourceID())

	assert.Equal(t, mapstr.M{
		"cloudfoundry": mapstr.M{
			"type": "log",
			"log": mapstr.M{
				"source": mapstr.M{
					"instance": evt.SourceID(),
					"type":     evt.SourceType(),
				},
			},
			"envelope": mapstr.M{
				"origin":     "origin",
				"deployment": "deployment",
				"ip":         "ip",
				"job":        "job",
				"index":      "index",
			},
			"app": mapstr.M{
				"id":   "f47ac10b-58cc-4372-a567-0e02b2c3d479",
				"name": "some-app",
			},
			"space": mapstr.M{
				"id":   "e1114e92-155c-11eb-ada9-27b81025a657",
				"name": "some-space",
			},
			"org": mapstr.M{
				"id":   "baeef1ba-155c-11eb-a1af-8f14964c35d2",
				"name": "some-org",
			},
			"tags": mapstr.M{
				"custom_tag": "foo",
			},
		},
		"message": "log message",
		"stream":  "stdout",
	}, evt.ToFields())
}

func makeEnvelope(eventType *events.Envelope_EventType) *events.Envelope {
	timestamp := int64(1587469726082)
	origin := "origin"
	deployment := "deployment"
	job := "job"
	index := "index"
	ip := "ip"
	return &events.Envelope{
		Origin:     &origin,
		EventType:  eventType,
		Timestamp:  &timestamp,
		Deployment: &deployment,
		Job:        &job,
		Index:      &index,
		Ip:         &ip,
		Tags:       map[string]string{"tag": "value"},
	}
}

func makeUUID() *events.UUID {
	// UUID `f47ac10b-58cc-4372-a567-0e02b2c3d479`
	low := uint64(0x7243cc580bc17af4)
	high := uint64(0x79d4c3b2020e67a5)
	return &events.UUID{
		Low:  &low,
		High: &high,
	}
}
