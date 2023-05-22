// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux

package synthexec

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
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
			u, _ := url.Parse(url1)
			// journey end gets a summary
			v = append(v, lookslike.MustCompile(map[string]interface{}{
				"event.type":          "heartbeat/summary",
				"synthetics.type":     "heartbeat/summary",
				"url":                 wrappers.URLFields(u),
				"monitor.duration.us": int64(journeyEnd.Timestamp().Sub(journeyStart.Timestamp()) / time.Microsecond),
				"monitor.check_group": isdef.IsString,
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

	tests := []struct {
		name                  string
		IsLegacyBrowserSource bool
	}{
		{
			name:                  "legacy project monitor",
			IsLegacyBrowserSource: true,
		},
		{
			name:                  "modern monitor",
			IsLegacyBrowserSource: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sFields.IsLegacyBrowserSource = tt.IsLegacyBrowserSource

			je := makeTestJourneyEnricher(sFields)
			for _, se := range synthEvents {
				check(t, se, je)
			}
		})
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
				v := lookslike.MustCompile(mapstr.M{
					"event": map[string]string{
						"type": "heartbeat/summary",
					},
				})
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
			true,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"event": map[string]string{
						"type": "heartbeat/summary",
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			"journey/end",
			&SynthEvent{Type: JourneyEnd},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"event": map[string]string{
						"type": "heartbeat/summary",
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			"step/end",
			&SynthEvent{Type: "step/end"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, 1, je.stepCount)
			},
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
			tt.check(t, e, je)
		})
	}
}

func TestNoSummaryOnAfterHook(t *testing.T) {
	journey := &Journey{
		Name: "A journey that fails after completing",
		ID:   "my-bad-after-all-hook",
	}
	journeyStart := &SynthEvent{
		Type:                 JourneyStart,
		TimestampEpochMicros: 1000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}
	syntherr := &SynthError{
		Message: "my-errmsg",
		Name:    "my-errname",
		Stack:   "my\nerr\nstack",
	}
	journeyEnd := &SynthEvent{
		Type:                 JourneyEnd,
		TimestampEpochMicros: 2000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}
	cmdStatus := &SynthEvent{
		Type:                 CmdStatus,
		Error:                &SynthError{Name: "cmdexit", Message: "cmd err msg"},
		TimestampEpochMicros: 3000,
	}

	badStepUrl := "https://example.com/bad-step"
	synthEvents := []*SynthEvent{
		journeyStart,
		makeStepEvent("step/start", 10, "Step1", 1, "", "", nil),
		makeStepEvent("step/end", 20, "Step1", 2, "failed", badStepUrl, syntherr),
		journeyEnd,
		cmdStatus,
	}

	stdFields := stdfields.StdMonitorFields{}
	je := makeTestJourneyEnricher(stdFields)
	for idx, se := range synthEvents {
		e := &beat.Event{}

		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se)

			if se != nil && se.Type == CmdStatus {
				t.Run("no summary in cmd/status", func(t *testing.T) {
					require.NotContains(t, e.Fields, "summary")
				})
			}

			// Only the journey/end event should get a summary when
			// it's emitted before the cmd/status (when an afterX hook fails).
			if se != nil && se.Type == JourneyEnd {
				require.Equal(t, stepError(syntherr), enrichErr)

				u, _ := url.Parse(badStepUrl)
				t.Run("summary in journey/end", func(t *testing.T) {
					v := lookslike.MustCompile(mapstr.M{
						"synthetics.type":     "heartbeat/summary",
						"url":                 wrappers.URLFields(u),
						"monitor.duration.us": int64(journeyEnd.Timestamp().Sub(journeyStart.Timestamp()) / time.Microsecond),
					})

					testslike.Test(t, v, e.Fields)
				})
			}
		})
	}
}

func TestSummaryWithoutJourneyEnd(t *testing.T) {
	journey := &Journey{
		Name: "A journey that never emits journey/end but exits successfully",
		ID:   "no-journey-end-but-success",
	}
	journeyStart := &SynthEvent{
		Type:                 "journey/start",
		TimestampEpochMicros: 1000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}

	cmdStatus := &SynthEvent{
		Type:                 CmdStatus,
		Error:                nil,
		TimestampEpochMicros: 3000,
	}

	url1 := "http://example.net/url1"
	synthEvents := []*SynthEvent{
		journeyStart,
		makeStepEvent("step/end", 20, "Step1", 1, "", url1, nil),
		cmdStatus,
	}

	hasCmdStatus := false

	stdFields := stdfields.StdMonitorFields{}
	je := makeTestJourneyEnricher(stdFields)
	for idx, se := range synthEvents {
		e := &beat.Event{}
		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se)

			if se != nil && se.Type == CmdStatus {
				hasCmdStatus = true
				require.Error(t, enrichErr, "journey did not finish executing, 1 steps ran")

				u, _ := url.Parse(url1)

				v := lookslike.MustCompile(mapstr.M{
					"synthetics.type":     "heartbeat/summary",
					"url":                 wrappers.URLFields(u),
					"monitor.duration.us": int64(cmdStatus.Timestamp().Sub(journeyStart.Timestamp()) / time.Microsecond),
				})

				testslike.Test(t, v, e.Fields)
			}
		})
	}

	require.True(t, hasCmdStatus)
}

func TestCreateSummaryEvent(t *testing.T) {
	baseTime := time.Now()

	testJourney := Journey{
		ID:   "my-monitor",
		Name: "My Monitor",
	}

	tests := []struct {
		name     string
		je       *journeyEnricher
		expected mapstr.M
		wantErr  bool
	}{{
		name: "completed without errors",
		je: &journeyEnricher{
			journey:         &testJourney,
			start:           baseTime,
			end:             baseTime.Add(10 * time.Microsecond),
			journeyComplete: true,
			stepCount:       3,
		},
		expected: mapstr.M{
			"monitor.duration.us": int64(10),
			"event": mapstr.M{
				"type": "heartbeat/summary",
			},
		},
		wantErr: false,
	}, {
		name: "completed with error",
		je: &journeyEnricher{
			journey:         &testJourney,
			start:           baseTime,
			end:             baseTime.Add(10 * time.Microsecond),
			journeyComplete: true,
			errorCount:      1,
			error:           fmt.Errorf("journey errored"),
		},
		expected: mapstr.M{
			"monitor.duration.us": int64(10),
			"event": mapstr.M{
				"type": "heartbeat/summary",
			},
		},
		wantErr: true,
	}, {
		name: "started, but exited without running steps",
		je: &journeyEnricher{
			journey:         &testJourney,
			start:           baseTime,
			end:             baseTime.Add(10 * time.Microsecond),
			stepCount:       0,
			journeyComplete: false,
			streamEnricher:  newStreamEnricher(stdfields.StdMonitorFields{}),
		},
		expected: mapstr.M{
			"monitor.duration.us": int64(10),
			"event": mapstr.M{
				"type": "heartbeat/summary",
			},
		},
		wantErr: true,
	}, {
		name: "syntax error - exited without starting",
		je: &journeyEnricher{
			journey:         &testJourney,
			end:             time.Now().Add(10 * time.Microsecond),
			journeyComplete: false,
			errorCount:      1,
			streamEnricher:  newStreamEnricher(stdfields.StdMonitorFields{}),
		},
		expected: mapstr.M{
			"event": mapstr.M{
				"type": "heartbeat/summary",
			},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitorField := mapstr.M{"id": "my-monitor", "type": "browser"}

			e := &beat.Event{
				Fields: mapstr.M{"monitor": monitorField},
			}
			err := tt.je.createSummary(e)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			// linter has been activated in the meantime. We'll cleanup separately.
			err = mapstr.MergeFields(tt.expected, mapstr.M{
				"monitor":            monitorField,
				"url":                mapstr.M{},
				"event.type":         "heartbeat/summary",
				"synthetics.type":    "heartbeat/summary",
				"synthetics.journey": testJourney,
			}, true)
			require.NoError(t, err)
			testslike.Test(t, lookslike.Strict(lookslike.MustCompile(tt.expected)), e.Fields)
		})
	}
}

func makeTestJourneyEnricher(sFields stdfields.StdMonitorFields) *journeyEnricher {
	return &journeyEnricher{
		streamEnricher: newStreamEnricher(sFields),
	}
}
