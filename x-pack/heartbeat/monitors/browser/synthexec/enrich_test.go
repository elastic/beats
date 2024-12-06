// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || synthetics

package synthexec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
	"github.com/elastic/go-lookslike/validator"
)

func makeStepEvent(typ string, ts float64, name string, index int, status string, urlstr string, err *SynthError) *SynthEvent {
	return &SynthEvent{
		Type:                 typ,
		TimestampEpochMicros: 1000 + ts,
		PackageVersion:       "1.0.0",
		Step:                 &Step{Name: name, Index: index, Status: status},
		Error:                err,
		Payload:              mapstr.M{},
		URL:                  urlstr,
	}
}

func TestJourneyEnricher(t *testing.T) {
	var sFields = stdfields.StdMonitorFields{
		ID:   "myproject",
		Name: "myproject",
		Type: "browser",
	}
	journey := &Journey{
		Name: "A Journey Name",
		ID:   "my-journey-id",
	}
	syntherr := &SynthError{
		Message: "my-errmsg",
		Name:    "my-errname",
		Stack:   "my\nerr\nstack",
	}
	otherErr := &SynthError{
		Message: "last-errmsg",
		Name:    "last-errname",
		Stack:   "last\nerr\nstack",
	}
	journeyStart := &SynthEvent{
		Type:                 JourneyStart,
		TimestampEpochMicros: 1000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}
	journeyEnd := &SynthEvent{
		Type:                 JourneyEnd,
		TimestampEpochMicros: 2000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}
	url1 := "http://example.net/url1"
	url2 := "http://example.net/url2"
	url3 := "http://example.net/url3"

	synthEvents := []*SynthEvent{
		journeyStart,
		makeStepEvent("step/start", 10, "Step1", 1, "succeeded", "", nil),
		makeStepEvent("step/end", 20, "Step1", 1, "", url1, nil),
		makeStepEvent("step/start", 21, "Step2", 2, "", "", nil),
		makeStepEvent("step/end", 30, "Step2", 2, "failed", url2, syntherr),
		makeStepEvent("step/start", 31, "Step3", 3, "", "", nil),
		makeStepEvent("step/end", 40, "Step3", 3, "", url3, otherErr),
		journeyEnd,
	}

	valid := func(se *SynthEvent) validator.Validator {
		var v []validator.Validator

		// We need an expectation for each input plus a final
		// expectation for the summary which comes on the nil data.
		if se.Type != JourneyEnd {
			// Test that the created event includes the mapped
			// version of the event
			v = append(v, lookslike.MustCompile(se.ToMap()))
		} else {
			v = append(v, lookslike.MustCompile(map[string]interface{}{
				"event.type":      "journey/end",
				"synthetics.type": "journey/end",
			}))
		}
		return lookslike.Compose(v...)
	}

	check := func(t *testing.T, se *SynthEvent, je *journeyEnricher) {
		e := &beat.Event{}
		t.Run(fmt.Sprintf("event: %s", se.Type), func(t *testing.T) {
			// we invoke the stream enricher's enrich function, which in turn calls the journey enricher
			// we do this because we want the check group set
			enrichErr := je.streamEnricher.enrich(e, se)
			if se.Error != nil {
				require.Equal(t, stepError(se.Error), enrichErr)
			}

			testslike.Test(t, valid(se), e.Fields)

			require.Equal(t, se.Timestamp().Unix(), e.Timestamp.Unix())
		})
	}

	je := makeTestJourneyEnricher(sFields)
	for _, se := range synthEvents {
		check(t, se, je)
	}
}

func TestEnrichConsoleSynthEvents(t *testing.T) {
	tests := []struct {
		name  string
		se    *SynthEvent
		check func(t *testing.T, e *beat.Event, je *journeyEnricher)
	}{
		{
			"stderr",
			&SynthEvent{
				Type: Stderr,
				Payload: mapstr.M{
					"message": "Error from synthetics",
				},
				PackageVersion: "1.0.0",
			},
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"synthetics": mapstr.M{
						"payload": mapstr.M{
							"message": "Error from synthetics",
						},
						"type":            Stderr,
						"package_version": "1.0.0",
						"index":           0,
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			"stdout",
			&SynthEvent{
				Type: Stdout,
				Payload: mapstr.M{
					"message": "debug output",
				},
				PackageVersion: "1.0.0",
			},
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"synthetics": mapstr.M{
						"payload": mapstr.M{
							"message": "debug output",
						},
						"type":            Stdout,
						"package_version": "1.0.0",
						"index":           0,
					},
				})
				testslike.Test(t, lookslike.Strict(v), e.Fields)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &beat.Event{}
			je := newJourneyEnricher(newStreamEnricher(stdfields.StdMonitorFields{}))
			err := je.enrichSynthEvent(e, tt.se)
			require.NoError(t, err)
			tt.check(t, e, je)
		})
	}
}

func TestEnrichSynthEvent(t *testing.T) {
	tests := []struct {
		name    string
		se      *SynthEvent
		wantErr bool
		check   func(t *testing.T, e *beat.Event, je *journeyEnricher)
	}{
		{
			"cmd/status - with error",
			&SynthEvent{
				Type:  CmdStatus,
				Error: &SynthError{Name: "cmdexit", Message: "cmd err msg"},
			},
			true,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			// If a journey did not emit `journey/end` but exited without
			// errors, we consider the journey to be up.
			"cmd/status - without error",
			&SynthEvent{
				Type:  CmdStatus,
				Error: nil,
			},
			false,
			nil,
		},
		{
			"journey/end",
			&SynthEvent{Type: JourneyEnd},
			false,
			nil,
		},
		{
			"step/end",
			&SynthEvent{Type: "step/end"},
			false,
			nil,
		},
		{
			"step/screenshot",
			&SynthEvent{Type: "step/screenshot"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "browser.screenshot", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
		{
			"step/screenshot_ref",
			&SynthEvent{Type: "step/screenshot_ref"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "browser.screenshot", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
		{
			"step/screenshot_block",
			&SynthEvent{Type: "screenshot/block", Id: "my_id"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "my_id", e.Meta["_id"])
				require.Equal(t, events.OpTypeCreate, e.Meta[events.FieldMetaOpType])
				require.Equal(t, "browser.screenshot", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
		{
			"journey/network_info",
			&SynthEvent{Type: "journey/network_info"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "browser.network", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			je := newJourneyEnricher(newStreamEnricher(stdfields.StdMonitorFields{}))
			e := &beat.Event{}
			if err := je.enrichSynthEvent(e, tt.se); (err == nil && tt.wantErr) || (err != nil && !tt.wantErr) {
				t.Errorf("journeyEnricher.enrichSynthEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				tt.check(t, e, je)
			}
		})
	}
}

func makeTestJourneyEnricher(sFields stdfields.StdMonitorFields) *journeyEnricher {
	return &journeyEnricher{
		streamEnricher: newStreamEnricher(sFields),
	}
}
