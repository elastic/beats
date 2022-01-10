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
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
)

func makeStepEvent(typ string, ts float64, name string, index int, status string, urlstr string, err *SynthError) *SynthEvent {
	return &SynthEvent{
		Type:                 typ,
		TimestampEpochMicros: 1000 + ts,
		PackageVersion:       "1.0.0",
		Step:                 &Step{Name: name, Index: index, Status: status},
		Error:                err,
		Payload:              common.MapStr{},
		URL:                  urlstr,
	}
}

func TestJourneyEnricher(t *testing.T) {
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
		Payload:              common.MapStr{},
	}
	journeyEnd := &SynthEvent{
		Type:                 "journey/end",
		TimestampEpochMicros: 2000,
		PackageVersion:       "1.0.0",
		Journey:              journey,
		Payload:              common.MapStr{},
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

	je := &journeyEnricher{}

	// We need an expectation for each input
	// plus a final expectation for the summary which comes
	// on the nil data.
	for idx, se := range synthEvents {
		e := &beat.Event{}
		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se)

			if se != nil && se.Type != "journey/end" {
				// Test that the created event includes the mapped
				// version of the event
				testslike.Test(t, lookslike.MustCompile(se.ToMap()), e.Fields)
				require.Equal(t, se.Timestamp().Unix(), e.Timestamp.Unix())

				if se.Error != nil {
					require.Equal(t, stepError(se.Error), enrichErr)
				}
			} else { // journey end gets a summary
				require.Equal(t, stepError(syntherr), enrichErr)

				u, _ := url.Parse(url1)
				t.Run("summary", func(t *testing.T) {
					v := lookslike.MustCompile(common.MapStr{
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

func TestEnrichSynthEvent(t *testing.T) {
	tests := []struct {
		name    string
		je      *journeyEnricher
		se      *SynthEvent
		wantErr bool
		check   func(t *testing.T, e *beat.Event, je *journeyEnricher)
	}{
		{
			"cmd/status",
			&journeyEnricher{},
			&SynthEvent{
				Type:  "cmd/status",
				Error: &SynthError{Name: "cmdexit", Message: "cmd err msg"},
			},
			true,
			func(t *testing.T, e *beat.Event, je *journeyEnricher) {
				v := lookslike.MustCompile(map[string]interface{}{
					"summary": map[string]int{
						"up":   0,
						"down": 1,
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
				v := lookslike.MustCompile(map[string]interface{}{
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
			if err := tt.je.enrichSynthEvent(e, tt.se); (err != nil) != tt.wantErr {
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
		Payload:              common.MapStr{},
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
		Payload:              common.MapStr{},
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
		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se)

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
					v := lookslike.MustCompile(common.MapStr{
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
