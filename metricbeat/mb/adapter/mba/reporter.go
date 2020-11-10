package mba

import (
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

type reporter interface {
	V1() mb.PushReporter
	V2() mb.PushReporterV2
}

type eventReporter struct {
	eventTransformer eventTransformer
	client           beat.Client
	cancel           v2.Canceler
	stats            *stats
}

type eventTransformer struct {
	inputName string
	namespace string
	metricset mb.MetricSet
	modifiers []mb.EventModifier
	start     time.Time //TODO
	periodic  bool
}

func (r *eventReporter) StartFetchTimer()      { r.eventTransformer.start = time.Now() }
func (r *eventReporter) V1() mb.PushReporter   { return (*reporterV1)(r) }
func (r *eventReporter) V2() mb.PushReporterV2 { return (*reporterV2)(r) }

type reporterV1 eventReporter

func (r *reporterV1) access() *eventReporter         { return (*eventReporter)(r) }
func (r *reporterV1) V2() *reporterV2                { return (*reporterV2)(r) }
func (r *reporterV1) Done() <-chan struct{}          { return r.access().cancel.Done() }
func (r *reporterV1) Event(event common.MapStr) bool { return r.ErrorWith(nil, event) }
func (r *reporterV1) Error(err error) bool           { return r.ErrorWith(err, nil) }
func (r *reporterV1) ErrorWith(err error, meta common.MapStr) bool {
	if err == nil && meta == nil {
		return true
	}
	return r.V2().Event(fromMapStr(r.access().eventTransformer.inputName, meta, err))
}

type reporterV2 eventReporter

func (r *reporterV2) access() *eventReporter { return (*eventReporter)(r) }
func (r *reporterV2) Error(err error) bool   { return r.Event(mb.Event{Error: err}) }
func (r *reporterV2) Done() <-chan struct{}  { return r.access().cancel.Done() }
func (r *reporterV2) Event(event mb.Event) bool {
	er := r.access()

	if event.Error == nil {
		r.stats.success.Add(1)
	} else {
		r.stats.failures.Add(1)
	}

	beatEvent := er.eventTransformer.BuildBeatsEvent(&event)
	if !sendEvent(r.cancel, r.client, beatEvent) {
		return false
	}
	er.stats.events.Add(1)

	return true
}

func sendEvent(cancel v2.Canceler, client beat.Client, event beat.Event) bool {
	if cancel.Err() != nil {
		return false
	}

	client.Publish(event)
	return cancel.Err() == nil
}

func (et *eventTransformer) BuildBeatsEvent(e *mb.Event) beat.Event {
	return et.finalizeEvent(e).BeatEvent(
		et.metricset.Module().Name(),
		et.metricset.Name(),
		et.modifiers...,
	)
}

// finalizeEvent adds missing fields that are not required by modules to be set
// and fixes some mappings.
func (et *eventTransformer) finalizeEvent(e *mb.Event) *mb.Event {
	if e.Took == 0 && !et.start.IsZero() {
		e.Took = time.Since(et.start)
	}
	if et.periodic {
		e.Period = et.metricset.Module().Config().Period
	}

	if e.Timestamp.IsZero() {
		if !et.start.IsZero() {
			e.Timestamp = et.start
		} else {
			e.Timestamp = time.Now().UTC()
		}
	}

	if e.Host == "" {
		e.Host = et.metricset.Host()
	}

	if e.Namespace == "" {
		e.Namespace = et.namespace
	}

	return e
}

// fromMapStr transforms a common.MapStr produced by MetricSet
// (like any MetricSet that does not natively produce a mb.Event). It accounts
// for the special key names and routes the data stored under those keys to the
// correct location in the event.
//
// XXX: this function is a modified copy of mb.TransformMapStrToEvent
func fromMapStr(module string, m common.MapStr, err error) mb.Event {
	var (
		event = mb.Event{RootFields: common.MapStr{}, Error: err}
	)

	for k, v := range m {
		switch k {
		case mb.TimestampKey:
			switch ts := v.(type) {
			case time.Time:
				delete(m, mb.TimestampKey)
				event.Timestamp = ts
			case common.Time:
				delete(m, mb.TimestampKey)
				event.Timestamp = time.Time(ts)
			}
		case mb.ModuleDataKey:
			delete(m, mb.ModuleDataKey)
			event.ModuleFields, _ = tryToMapStr(v)
		case mb.RTTKey:
			delete(m, mb.RTTKey)
			if took, ok := v.(time.Duration); ok {
				event.Took = took
			}
		case mb.NamespaceKey:
			delete(m, mb.NamespaceKey)
			event.Namespace = module
		}
	}

	event.MetricSetFields = m
	return event
}

func tryToMapStr(v interface{}) (common.MapStr, bool) {
	switch m := v.(type) {
	case common.MapStr:
		return m, true
	case map[string]interface{}:
		return common.MapStr(m), true
	default:
		return nil, false
	}
}
