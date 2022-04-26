// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
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
	var stdFields = StdSuiteFields{
		Id:       "mysuite",
		Name:     "mysuite",
		Type:     "browser",
		IsInline: false,
	}
	journey := &Journey{
		Name: "A Journey Name",
		Id:   "my-journey-id",
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
		Type:                 "journey/start",
		TimestampEpochMicros: 1000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}
	journeyEnd := &SynthEvent{
		Type:                 "journey/end",
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
		makeStepEvent("step/start", 21, "Step2", 1, "", "", nil),
		makeStepEvent("step/end", 30, "Step2", 1, "failed", url2, syntherr),
		makeStepEvent("step/start", 31, "Step3", 1, "", "", nil),
		makeStepEvent("step/end", 40, "Step3", 1, "", url3, otherErr),
		journeyEnd,
	}

	suiteValidator := func() validator.Validator {
		return lookslike.MustCompile(mapstr.M{
			"suite.id":     stdFields.Id,
			"suite.name":   stdFields.Name,
			"monitor.id":   fmt.Sprintf("%s-%s", stdFields.Id, journey.Id),
			"monitor.name": fmt.Sprintf("%s - %s", stdFields.Name, journey.Name),
			"monitor.type": stdFields.Type,
		})
	}
	inlineValidator := func() validator.Validator {
		return lookslike.MustCompile(mapstr.M{
			"monitor.id":   stdFields.Id,
			"monitor.name": stdFields.Name,
			"monitor.type": stdFields.Type,
		})
	}
	commonValidator := func(se *SynthEvent) validator.Validator {
		var v []validator.Validator

		// We need an expectation for each input plus a final
		// expectation for the summary which comes on the nil data.
		if se.Type != "journey/end" {
			// Test that the created event includes the mapped
			// version of the event
			v = append(v, lookslike.MustCompile(se.ToMap()))
		} else {
			u, _ := url.Parse(url1)
			// journey end gets a summary
			v = append(v, lookslike.MustCompile(mapstr.M{
				"synthetics.type":     "heartbeat/summary",
				"url":                 wrappers.URLFields(u),
				"monitor.duration.us": int64(journeyEnd.Timestamp().Sub(journeyStart.Timestamp()) / time.Microsecond),
			}))
		}
		return lookslike.Compose(v...)
	}

	je := &journeyEnricher{}
	check := func(t *testing.T, se *SynthEvent, ssf StdSuiteFields) {
		e := &beat.Event{}
		t.Run(fmt.Sprintf("event: %s", se.Type), func(t *testing.T) {
			enrichErr := je.enrich(e, se, ssf)
			if se.Error != nil {
				require.Equal(t, stepError(se.Error), enrichErr)
			}
			if ssf.IsInline {
				sv, _ := e.Fields.GetValue("suite")
				require.Nil(t, sv)
				testslike.Test(t, inlineValidator(), e.Fields)
			} else {
				testslike.Test(t, suiteValidator(), e.Fields)
			}
			testslike.Test(t, commonValidator(se), e.Fields)

			require.Equal(t, se.Timestamp().Unix(), e.Timestamp.Unix())
		})
	}

	tests := []struct {
		name     string
		isInline bool
	}{
		{
			name:     "suite monitor",
			isInline: false,
		},
		{
			name:     "inline monitor",
			isInline: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdFields.IsInline = tt.isInline
			for _, se := range synthEvents {
				check(t, se, stdFields)
			}
		})
	}
}

func TestEnrichConsoleSynthEvents(t *testing.T) {
	tests := []struct {
		name  string
		je    *journeyEnricher
		se    *SynthEvent
		check func(t *testing.T, e *beat.Event, je *journeyEnricher)
	}{
		{
			"stderr",
			&journeyEnricher{},
			&SynthEvent{
				Type: "stderr",
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
						"type":            "stderr",
						"package_version": "1.0.0",
						"index":           0,
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			"stdout",
			&journeyEnricher{},
			&SynthEvent{
				Type: "stdout",
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
						"type":            "stdout",
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
			tt.je.enrichSynthEvent(e, tt.se)
			tt.check(t, e, tt.je)
		})
	}
}

func TestEnrichSynthEvent(t *testing.T) {
	tests := []struct {
		name    string
		je      *journeyEnricher
		se      *SynthEvent
		wantErr bool
		check   func(t *testing.T, e *beat.Event, je *journeyEnricher)
	}{
		{
			"cmd/status - with error",
			&journeyEnricher{},
			&SynthEvent{
				Type:  "cmd/status",
				Error: &SynthError{Name: "cmdexit", Message: "cmd err msg"},
			},
			true,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"summary": map[string]int{
						"up":   0,
						"down": 1,
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			// If a journey did not emit `journey/end` but exited without
			// errors, we consider the journey to be up.
			"cmd/status - without error",
			&journeyEnricher{},
			&SynthEvent{
				Type:  "cmd/status",
				Error: nil,
			},
			true,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"summary": map[string]int{
						"up":   1,
						"down": 0,
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			"journey/end",
			&journeyEnricher{},
			&SynthEvent{Type: "journey/end"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(mapstr.M{
					"summary": map[string]int{
						"up":   1,
						"down": 0,
					},
				})
				testslike.Test(t, v, e.Fields)
			},
		},
		{
			"step/end",
			&journeyEnricher{},
			&SynthEvent{Type: "step/end"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, 1, je.stepCount)
			},
		},
		{
			"step/screenshot",
			&journeyEnricher{},
			&SynthEvent{Type: "step/screenshot"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "browser.screenshot", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
		{
			"step/screenshot_ref",
			&journeyEnricher{},
			&SynthEvent{Type: "step/screenshot_ref"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "browser.screenshot", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
		{
			"step/screenshot_block",
			&journeyEnricher{},
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
			&journeyEnricher{},
			&SynthEvent{Type: "journey/network_info"},
			false,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				require.Equal(t, "browser.network", e.Meta[add_data_stream.FieldMetaCustomDataset])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &beat.Event{}
			if err := tt.je.enrichSynthEvent(e, tt.se); (err == nil && tt.wantErr) || (err != nil && !tt.wantErr) {
				t.Errorf("journeyEnricher.enrichSynthEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.check(t, e, tt.je)
		})
	}
}

func TestNoSummaryOnAfterHook(t *testing.T) {
	journey := &Journey{
		Name: "A journey that fails after completing",
		Id:   "my-bad-after-all-hook",
	}
	journeyStart := &SynthEvent{
		Type:                 "journey/start",
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
		Type:                 "journey/end",
		TimestampEpochMicros: 2000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}
	cmdStatus := &SynthEvent{
		Type:                 "cmd/status",
		Error:                &SynthError{Name: "cmdexit", Message: "cmd err msg"},
		TimestampEpochMicros: 3000,
	}

	badStepUrl := "https://example.com/bad-step"
	synthEvents := []*SynthEvent{
		journeyStart,
		makeStepEvent("step/start", 10, "Step1", 1, "", "", nil),
		makeStepEvent("step/end", 20, "Step1", 1, "failed", badStepUrl, syntherr),
		journeyEnd,
		cmdStatus,
	}

	je := &journeyEnricher{}

	for idx, se := range synthEvents {
		e := &beat.Event{}
		stdFields := StdSuiteFields{IsInline: false}
		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se, stdFields)

			if se != nil && se.Type == "cmd/status" {
				t.Run("no summary in cmd/status", func(t *testing.T) {
					require.NotContains(t, e.Fields, "summary")
				})
			}

			// Only the journey/end event should get a summary when
			// it's emitted before the cmd/status (when an afterX hook fails).
			if se != nil && se.Type == "journey/end" {
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
		Id:   "no-journey-end-but-success",
	}
	journeyStart := &SynthEvent{
		Type:                 "journey/start",
		TimestampEpochMicros: 1000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              mapstr.M{},
	}

	cmdStatus := &SynthEvent{
		Type:                 "cmd/status",
		Error:                nil,
		TimestampEpochMicros: 3000,
	}

	url1 := "http://example.net/url1"
	synthEvents := []*SynthEvent{
		journeyStart,
		makeStepEvent("step/end", 20, "Step1", 1, "", url1, nil),
		cmdStatus,
	}

	je := &journeyEnricher{}

	hasCmdStatus := false

	for idx, se := range synthEvents {
		e := &beat.Event{}
		stdFields := StdSuiteFields{IsInline: false}
		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se, stdFields)

			if se != nil && se.Type == "cmd/status" {
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
	tests := []struct {
		name     string
		je       *journeyEnricher
		expected mapstr.M
		wantErr  bool
	}{{
		name: "completed without errors",
		je: &journeyEnricher{
			journey:         &Journey{},
			start:           time.Now(),
			end:             time.Now().Add(10 * time.Microsecond),
			journeyComplete: true,
		},
		expected: mapstr.M{
			"monitor.duration.us": int64(10),
			"summary": mapstr.M{
				"down": 0,
				"up":   1,
			},
		},
		wantErr: false,
	}, {
		name: "completed with error",
		je: &journeyEnricher{
			journey:         &Journey{},
			start:           time.Now(),
			end:             time.Now().Add(10 * time.Microsecond),
			journeyComplete: true,
			errorCount:      1,
			firstError:      fmt.Errorf("journey errored"),
		},
		expected: mapstr.M{
			"monitor.duration.us": int64(10),
			"summary": mapstr.M{
				"down": 1,
				"up":   0,
			},
		},
		wantErr: true,
	}, {
		name: "started, but exited without running steps",
		je: &journeyEnricher{
			journey:         &Journey{},
			start:           time.Now(),
			end:             time.Now().Add(10 * time.Microsecond),
			journeyComplete: false,
		},
		expected: mapstr.M{
			"monitor.duration.us": int64(10),
			"summary": mapstr.M{
				"down": 0,
				"up":   1,
			},
		},
		wantErr: true,
	}, {
		name: "syntax error - exited without starting",
		je: &journeyEnricher{
			journey:         &Journey{},
			end:             time.Now().Add(10 * time.Microsecond),
			journeyComplete: false,
			errorCount:      1,
		},
		expected: mapstr.M{
			"summary": mapstr.M{
				"down": 1,
				"up":   0,
			},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &beat.Event{}
			err := tt.je.createSummary(e)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			common.MergeFields(tt.expected, mapstr.M{
				"url":                mapstr.M{},
				"event.type":         "heartbeat/summary",
				"synthetics.type":    "heartbeat/summary",
				"synthetics.journey": Journey{},
			}, true)
			testslike.Test(t, lookslike.Strict(lookslike.MustCompile(tt.expected)), e.Fields)
		})
	}
}
