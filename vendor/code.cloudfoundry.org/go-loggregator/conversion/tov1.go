package conversion

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

// ToV1 converts v2 envelopes down to v1 envelopes. The v2 Envelope may be
// mutated during the conversion and share pointers with the new v1 envelope
// for efficiency in creating the v1 envelope. As a result the envelope you
// pass in should no longer be used.
func ToV1(e *loggregator_v2.Envelope) []*events.Envelope {
	var envelopes []*events.Envelope
	switch (e.Message).(type) {
	case *loggregator_v2.Envelope_Log:
		envelopes = convertLog(e)
	case *loggregator_v2.Envelope_Counter:
		envelopes = convertCounter(e)
	case *loggregator_v2.Envelope_Gauge:
		envelopes = convertGauge(e)
	case *loggregator_v2.Envelope_Timer:
		envelopes = convertTimer(e)
	}

	for _, v1e := range envelopes {
		delete(v1e.Tags, "__v1_type")
		delete(v1e.Tags, "origin")
		delete(v1e.Tags, "deployment")
		delete(v1e.Tags, "job")
		delete(v1e.Tags, "index")
		delete(v1e.Tags, "ip")
	}

	return envelopes
}

func createBaseV1(e *loggregator_v2.Envelope) *events.Envelope {
	v1e := &events.Envelope{
		Origin:     proto.String(getV2Tag(e, "origin")),
		Deployment: proto.String(getV2Tag(e, "deployment")),
		Job:        proto.String(getV2Tag(e, "job")),
		Index:      proto.String(getV2Tag(e, "index")),
		Timestamp:  proto.Int64(e.Timestamp),
		Ip:         proto.String(getV2Tag(e, "ip")),
		Tags:       convertTags(e),
	}

	if e.SourceId != "" {
		v1e.Tags["source_id"] = e.SourceId
	}

	return v1e
}

func getV2Tag(e *loggregator_v2.Envelope, key string) string {
	if value, ok := e.GetTags()[key]; ok {
		return value
	}

	d := e.GetDeprecatedTags()[key]
	if d == nil {
		return ""
	}

	switch v := d.Data.(type) {
	case *loggregator_v2.Value_Text:
		return v.Text
	case *loggregator_v2.Value_Integer:
		return fmt.Sprintf("%d", v.Integer)
	case *loggregator_v2.Value_Decimal:
		return fmt.Sprintf("%f", v.Decimal)
	default:
		return ""
	}
}

func convertTimer(v2e *loggregator_v2.Envelope) []*events.Envelope {
	v1e := createBaseV1(v2e)
	timer := v2e.GetTimer()
	v1e.EventType = events.Envelope_HttpStartStop.Enum()
	instanceIndex, err := strconv.Atoi(v2e.InstanceId)
	if err != nil {
		instanceIndex = 0
	}

	method := events.Method(events.Method_value[getV2Tag(v2e, "method")])
	peerType := events.PeerType(events.PeerType_value[getV2Tag(v2e, "peer_type")])

	v1e.HttpStartStop = &events.HttpStartStop{
		StartTimestamp: proto.Int64(timer.Start),
		StopTimestamp:  proto.Int64(timer.Stop),
		RequestId:      convertUUID(parseUUID(getV2Tag(v2e, "request_id"))),
		ApplicationId:  convertUUID(parseUUID(v2e.SourceId)),
		PeerType:       &peerType,
		Method:         &method,
		Uri:            proto.String(getV2Tag(v2e, "uri")),
		RemoteAddress:  proto.String(getV2Tag(v2e, "remote_address")),
		UserAgent:      proto.String(getV2Tag(v2e, "user_agent")),
		StatusCode:     proto.Int32(int32(atoi(getV2Tag(v2e, "status_code")))),
		ContentLength:  proto.Int64(atoi(getV2Tag(v2e, "content_length"))),
		InstanceIndex:  proto.Int32(int32(instanceIndex)),
		InstanceId:     proto.String(getV2Tag(v2e, "routing_instance_id")),
		Forwarded:      strings.Split(getV2Tag(v2e, "forwarded"), "\n"),
	}

	delete(v1e.Tags, "peer_type")
	delete(v1e.Tags, "method")
	delete(v1e.Tags, "request_id")
	delete(v1e.Tags, "uri")
	delete(v1e.Tags, "remote_address")
	delete(v1e.Tags, "user_agent")
	delete(v1e.Tags, "status_code")
	delete(v1e.Tags, "content_length")
	delete(v1e.Tags, "routing_instance_id")
	delete(v1e.Tags, "forwarded")

	return []*events.Envelope{v1e}
}

func atoi(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}

	return i
}

func convertLog(v2e *loggregator_v2.Envelope) []*events.Envelope {
	v1e := createBaseV1(v2e)
	if getV2Tag(v2e, "__v1_type") == "Error" {
		recoverError(v1e, v2e)
		return []*events.Envelope{v1e}
	}
	logMessage := v2e.GetLog()
	v1e.EventType = events.Envelope_LogMessage.Enum()
	v1e.LogMessage = &events.LogMessage{
		Message:        logMessage.Payload,
		MessageType:    messageType(logMessage),
		Timestamp:      proto.Int64(v2e.Timestamp),
		AppId:          proto.String(v2e.SourceId),
		SourceType:     proto.String(getV2Tag(v2e, "source_type")),
		SourceInstance: proto.String(v2e.InstanceId),
	}
	delete(v1e.Tags, "source_type")

	return []*events.Envelope{v1e}
}

func recoverError(v1e *events.Envelope, v2e *loggregator_v2.Envelope) {
	logMessage := v2e.GetLog()
	v1e.EventType = events.Envelope_Error.Enum()
	code := int32(atoi(getV2Tag(v2e, "code")))
	v1e.Error = &events.Error{
		Source:  proto.String(getV2Tag(v2e, "source")),
		Code:    proto.Int32(code),
		Message: proto.String(string(logMessage.Payload)),
	}
	delete(v1e.Tags, "source")
	delete(v1e.Tags, "code")
}

func convertCounter(v2e *loggregator_v2.Envelope) []*events.Envelope {
	v1e := createBaseV1(v2e)
	counterEvent := v2e.GetCounter()
	v1e.EventType = events.Envelope_CounterEvent.Enum()
	if v2e.InstanceId != "" {
		v1e.GetTags()["instance_id"] = v2e.InstanceId
	}
	v1e.CounterEvent = &events.CounterEvent{
		Name:  proto.String(counterEvent.Name),
		Delta: proto.Uint64(counterEvent.GetDelta()),
		Total: proto.Uint64(counterEvent.GetTotal()),
	}

	return []*events.Envelope{v1e}
}

func convertGauge(v2e *loggregator_v2.Envelope) []*events.Envelope {
	if v1e := tryConvertContainerMetric(v2e); v1e != nil {
		return []*events.Envelope{v1e}
	}

	var results []*events.Envelope
	gaugeEvent := v2e.GetGauge()

	for key, metric := range gaugeEvent.Metrics {
		v1e := createBaseV1(v2e)
		v1e.EventType = events.Envelope_ValueMetric.Enum()
		unit, value, ok := extractGaugeValues(metric)
		if !ok {
			return nil
		}

		if v2e.InstanceId != "" {
			v1e.GetTags()["instance_id"] = v2e.InstanceId
		}
		v1e.ValueMetric = &events.ValueMetric{
			Name:  proto.String(key),
			Unit:  proto.String(unit),
			Value: proto.Float64(value),
		}
		results = append(results, v1e)
	}

	return results
}

func extractGaugeValues(metric *loggregator_v2.GaugeValue) (string, float64, bool) {
	if metric == nil {
		return "", 0, false
	}

	return metric.Unit, metric.Value, true
}

func instanceIndex(v2e *loggregator_v2.Envelope) int32 {
	defaultIndex, err := strconv.Atoi(v2e.InstanceId)
	if err != nil {
		defaultIndex = 0
	}

	id := v2e.GetGauge().GetMetrics()["instance_index"]
	if id == nil {
		return int32(defaultIndex)
	}
	return int32(id.Value)
}

func tryConvertContainerMetric(v2e *loggregator_v2.Envelope) *events.Envelope {
	v1e := createBaseV1(v2e)
	gaugeEvent := v2e.GetGauge()
	if len(gaugeEvent.Metrics) == 1 {
		return nil
	}

	required := []string{
		"cpu",
		"memory",
		"disk",
		"memory_quota",
		"disk_quota",
	}

	for _, req := range required {
		if v, ok := gaugeEvent.Metrics[req]; !ok || v == nil || (v.Unit == "" && v.Value == 0) {
			return nil
		}
	}

	v1e.EventType = events.Envelope_ContainerMetric.Enum()
	v1e.ContainerMetric = &events.ContainerMetric{
		ApplicationId:    proto.String(v2e.SourceId),
		InstanceIndex:    proto.Int32(instanceIndex(v2e)),
		CpuPercentage:    proto.Float64(gaugeEvent.Metrics["cpu"].Value),
		MemoryBytes:      proto.Uint64(uint64(gaugeEvent.Metrics["memory"].Value)),
		DiskBytes:        proto.Uint64(uint64(gaugeEvent.Metrics["disk"].Value)),
		MemoryBytesQuota: proto.Uint64(uint64(gaugeEvent.Metrics["memory_quota"].Value)),
		DiskBytesQuota:   proto.Uint64(uint64(gaugeEvent.Metrics["disk_quota"].Value)),
	}

	return v1e
}

func convertTags(e *loggregator_v2.Envelope) map[string]string {
	oldTags := make(map[string]string)
	for k, v := range e.Tags {
		oldTags[k] = v
	}

	for key, value := range e.GetDeprecatedTags() {
		if value == nil {
			continue
		}
		switch value.Data.(type) {
		case *loggregator_v2.Value_Text:
			oldTags[key] = value.GetText()
		case *loggregator_v2.Value_Integer:
			oldTags[key] = fmt.Sprintf("%d", value.GetInteger())
		case *loggregator_v2.Value_Decimal:
			oldTags[key] = fmt.Sprintf("%f", value.GetDecimal())
		}
	}

	return oldTags
}

func messageType(log *loggregator_v2.Log) *events.LogMessage_MessageType {
	if log.Type == loggregator_v2.Log_OUT {
		return events.LogMessage_OUT.Enum()
	}
	return events.LogMessage_ERR.Enum()
}

func parseUUID(id string) []byte {
	// e.g. b3015d69-09cd-476d-aace-ad2d824d5ab7
	if len(id) != 36 {
		return nil
	}
	h := id[:8] + id[9:13] + id[14:18] + id[19:23] + id[24:]

	data, err := hex.DecodeString(h)
	if err != nil {
		return nil
	}

	return data
}

func convertUUID(id []byte) *events.UUID {
	if len(id) != 16 {
		return nil
	}

	return &events.UUID{
		Low:  proto.Uint64(binary.LittleEndian.Uint64(id[:8])),
		High: proto.Uint64(binary.LittleEndian.Uint64(id[8:])),
	}
}
