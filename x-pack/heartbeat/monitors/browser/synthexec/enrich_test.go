// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || synthetics

package synthexec

import (
	"encoding/json"
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

// API journeys must route network events to api.network, not browser.network.
func TestEnrichAPIJourneyDatasetRouting(t *testing.T) {
	se := newStreamEnricher(stdfields.StdMonitorFields{Type: "api"})
	// Prime the journey context via journey/start before the network_info event.
	startEvt := &beat.Event{}
	require.NoError(t, se.enrich(startEvt, &SynthEvent{
		Type:    JourneyStart,
		Journey: &Journey{ID: "j1", Name: "API journey", Type: "api"},
	}))

	netEvt := &beat.Event{}
	require.NoError(t, se.enrich(netEvt, &SynthEvent{Type: JourneyNetworkInfo}))

	require.Equal(t,
		"api.network",
		netEvt.Meta[add_data_stream.FieldMetaCustomDataset],
		"API journey/network_info must land in api.network (matching the Fleet integration's `data_stream/api_network` package), not browser.network",
	)
}

// Older agents (pre-apiJourney) omit journey.type; keep treating those as browser.
func TestEnrichLegacyJourneyDefaultsToBrowser(t *testing.T) {
	se := newStreamEnricher(stdfields.StdMonitorFields{Type: "browser"})
	startEvt := &beat.Event{}
	require.NoError(t, se.enrich(startEvt, &SynthEvent{
		Type:    JourneyStart,
		Journey: &Journey{ID: "j1", Name: "legacy"}, // no Type
	}))

	netEvt := &beat.Event{}
	require.NoError(t, se.enrich(netEvt, &SynthEvent{Type: JourneyNetworkInfo}))

	require.Equal(t,
		"browser.network",
		netEvt.Meta[add_data_stream.FieldMetaCustomDataset],
	)
}

// TestE2EAPIJourneyAgentOutput runs the exact ndjson an apiJourney emits
// (captured from elastic/synthetics#997) through the production decode path and
// the enricher, pinning the on-the-wire contract the struct-built tests skip.
func TestE2EAPIJourneyAgentOutput(t *testing.T) {
	// Shapes mirror the agent's json reporter, including the API-only server block.
	lines := []string{
		`{"@timestamp":1000000,"type":"journey/start","package_version":"1.30.0","journey":{"name":"Orders API","id":"orders-api","type":"api"},"payload":{"source":"apiJourney('Orders API', () => {})"}}`,
		`{"@timestamp":1500000,"type":"journey/network_info","package_version":"1.30.0","journey":{"name":"Orders API","id":"orders-api","type":"api"},"root_fields":{"url":"https://api.example.com/orders","user_agent":{"name":"api","version":""},"http":{"request":{"method":"GET","bytes":42},"response":{"status":200,"mime_type":"application/json","bytes":128}},"server":{"ip":"93.184.216.34","port":443}},"step":{"name":"list orders","index":1,"status":"succeeded"},"payload":{"type":"fetch","is_navigation_request":false,"transfer_size":128}}`,
		`{"@timestamp":2000000,"type":"journey/end","package_version":"1.30.0","journey":{"name":"Orders API","id":"orders-api","type":"api"},"payload":{"status":"succeeded"}}`,
	}

	senr := newStreamEnricher(stdfields.StdMonitorFields{ID: "orders-api", Name: "Orders API", Type: "api"})

	evts := make([]*beat.Event, len(lines))
	for i, line := range lines {
		var se SynthEvent
		require.NoErrorf(t, json.Unmarshal([]byte(line), &se), "agent line %d must decode into SynthEvent", i)
		require.NotEmptyf(t, se.Type, "decoded agent line %d must have a type", i)

		e := &beat.Event{}
		require.NoErrorf(t, senr.enrich(e, &se), "enriching agent line %d must not error", i)
		evts[i] = e
	}
	startEvt, netEvt, endEvt := evts[0], evts[1], evts[2]

	jType, err := startEvt.Fields.GetValue("synthetics.journey.type")
	require.NoError(t, err, "journey.type from the agent must propagate")
	require.Equal(t, "api", jType, "synthetics.journey.type must be api so downstream can distinguish API journeys")

	require.Equal(t,
		"api.network",
		netEvt.Meta[add_data_stream.FieldMetaCustomDataset],
		"apiJourney network events must route to api.network, matching the Fleet integration's api_network data stream",
	)

	ip, err := netEvt.Fields.GetValue("server.ip")
	require.NoError(t, err, "API-only server.ip from root_fields must survive enrichment")
	require.Equal(t, "93.184.216.34", ip, "server.ip must match the agent's captured remote address")
	port, err := netEvt.Fields.GetValue("server.port")
	require.NoError(t, err, "API-only server.port from root_fields must survive enrichment")
	require.EqualValues(t, 443, port, "server.port must match the agent's captured remote port")

	domain, err := netEvt.Fields.GetValue("url.domain")
	require.NoError(t, err, "the url string in root_fields must be parsed into ECS url.*")
	require.Equal(t, "api.example.com", domain, "url.domain must be derived from the network event url")

	for _, tc := range []struct {
		e    *beat.Event
		want string
	}{
		{startEvt, JourneyStart},
		{netEvt, JourneyNetworkInfo},
		{endEvt, JourneyEnd},
	} {
		st, err := tc.e.Fields.GetValue("synthetics.type")
		require.NoErrorf(t, err, "synthetics.type must be set for %s", tc.want)
		require.Equal(t, tc.want, st, "synthetics.type must reflect the agent event type")
	}
}

func makeTestJourneyEnricher(sFields stdfields.StdMonitorFields) *journeyEnricher {
	return &journeyEnricher{
		streamEnricher: newStreamEnricher(sFields),
	}
}
