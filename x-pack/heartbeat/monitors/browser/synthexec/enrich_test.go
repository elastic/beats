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
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/testslike"
)

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
	makeStepEvent := func(typ string, ts float64, name string, index int, urlstr string, err *SynthError) *SynthEvent {
		return &SynthEvent{
			Type:                 typ,
			TimestampEpochMicros: 1000 + ts,
			PackageVersion:       "1.0.0",
			Step:                 &Step{Name: name, Index: index},
			Error:                err,
			Payload:              common.MapStr{},
			URL:                  urlstr,
		}
	}
	url1 := "http://example.net/url1"
	url2 := "http://example.net/url2"
	url3 := "http://example.net/url3"

	synthEvents := []*SynthEvent{
		journeyStart,
		makeStepEvent("step/start", 10, "Step1", 1, "", nil),
		makeStepEvent("step/end", 20, "Step1", 1, url1, nil),
		makeStepEvent("step/start", 21, "Step2", 1, "", nil),
		makeStepEvent("step/end", 30, "Step2", 1, url2, syntherr),
		makeStepEvent("step/start", 31, "Step3", 1, "", nil),
		makeStepEvent("step/end", 40, "Step3", 1, url3, nil),
		journeyEnd,
	}

	je := &journeyEnricher{}

	// We need an expectation for each input
	// plus a final expectation for the summary which comes
	// on the nil data.
	for idx, se := range append(synthEvents, nil) {
		e := &beat.Event{}
		t.Run(fmt.Sprintf("event %d", idx), func(t *testing.T) {
			enrichErr := je.enrich(e, se)

			if se != nil {
				// Test that the created event includes the mapped
				// version of the event
				testslike.Test(t, lookslike.MustCompile(se.ToMap()), e.Fields)
				require.Equal(t, se.Timestamp().Unix(), e.Timestamp.Unix())

				if se.Error != nil {
					require.Equal(t, stepError(se.Error), enrichErr)
				}
			} else {
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
