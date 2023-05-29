// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin

package synthexec

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/logger"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type enricher func(event *beat.Event, se *SynthEvent) error

type streamEnricher struct {
	je           *journeyEnricher
	journeyCount int
	sFields      stdfields.StdMonitorFields
	checkGroup   string
}

func newStreamEnricher(sFields stdfields.StdMonitorFields) *streamEnricher {
	return &streamEnricher{sFields: sFields, checkGroup: makeUuid()}
}

func (senr *streamEnricher) enrich(event *beat.Event, se *SynthEvent) error {
	if senr.je == nil || (se != nil && se.Type == JourneyStart) {
		senr.je = newJourneyEnricher(senr)
	}

	// TODO: Remove this when zip monitors are removed and we have 1:1 monitor / journey
	if se != nil && se.Type == JourneyStart {
		senr.journeyCount++
		if senr.journeyCount > 1 {
			senr.checkGroup = makeUuid()
		}
	}

	eventext.MergeEventFields(event, map[string]interface{}{"monitor": map[string]interface{}{"check_group": senr.checkGroup}})
	return senr.je.enrich(event, se)
}

// journeyEnricher holds state across received SynthEvents retaining fields
// where relevant to properly enrich *beat.Event instances.
type journeyEnricher struct {
	journeyComplete bool
	journey         *Journey
	errorCount      int
	error           error
	stepCount       int
	// The first URL we visit is the URL for this journey, which is set on the summary event.
	// We store the URL fields here for use on the summary event.
	urlFields      mapstr.M
	start          time.Time
	end            time.Time
	streamEnricher *streamEnricher
}

func newJourneyEnricher(senr *streamEnricher) *journeyEnricher {
	return &journeyEnricher{
		streamEnricher: senr,
	}
}

func makeUuid() string {
	u, err := uuid.NewV1()
	if err != nil {
		panic("Cannot generate v1 UUID, this should never happen!")
	}
	return u.String()
}

func (je *journeyEnricher) enrich(event *beat.Event, se *SynthEvent) error {
	if se == nil {
		return nil
	}

	if !se.Timestamp().IsZero() {
		event.Timestamp = se.Timestamp()
		// Record start and end so we can calculate journey duration accurately later
		switch se.Type {
		case JourneyStart:
			je.error = nil
			je.journey = se.Journey
			je.start = event.Timestamp
		case JourneyEnd, CmdStatus:
			je.end = event.Timestamp
		}
	} else {
		event.Timestamp = time.Now()
	}

	eventext.MergeEventFields(event, mapstr.M{
		"event": mapstr.M{"type": se.Type},
	})

	return je.enrichSynthEvent(event, se)
}

func (je *journeyEnricher) enrichSynthEvent(event *beat.Event, se *SynthEvent) error {
	var jobErr error
	if se.Error != nil {
		jobErr = stepError(se.Error)
		if je.error == nil {
			je.error = jobErr
		}
	}

	// Needed for the edge case where a console log is emitted after one journey ends
	// but before another begins.
	if je.journey != nil {
		eventext.MergeEventFields(event, mapstr.M{
			"monitor": mapstr.M{
				"id":   je.journey.ID,
				"name": je.journey.Name,
			},
		})
	}

	switch se.Type {
	case CmdStatus:
		// If a command failed _after_ the journey was complete, as it happens
		// when an `afterAll` hook fails, for example, we don't wan't to include
		// a summary in the cmd/status event.
		if !je.journeyComplete {
			if se.Error != nil {
				je.error = se.Error.toECSErr()
			}
			return je.createSummary(event)
		}
	case JourneyEnd:
		je.journeyComplete = true
		return je.createSummary(event)
	case StepEnd:
		je.stepCount++
	case StepScreenshot, StepScreenshotRef, ScreenshotBlock:
		add_data_stream.SetEventDataset(event, "browser.screenshot")
	case JourneyNetworkInfo:
		add_data_stream.SetEventDataset(event, "browser.network")
	}

	if se.Id != "" {
		event.SetID(se.Id)
		// This is only relevant for screenshots, which have a specific ID
		// In that case we always want to issue an update op
		eventext.SetMeta(event, events.FieldMetaOpType, events.OpTypeCreate)
	}

	eventext.MergeEventFields(event, se.ToMap())

	if len(je.urlFields) == 0 {
		if urlFields, err := event.GetValue("url"); err == nil {
			if ufMap, ok := urlFields.(mapstr.M); ok {
				je.urlFields = ufMap
			}
		}
	}
	return jobErr
}

func (je *journeyEnricher) createSummary(event *beat.Event) error {
	// In case of syntax errors or incorrect runner options, the Synthetics
	// runner would exit immediately with exitCode 1 and we do not set the duration
	// to inform the journey never ran
	if !je.start.IsZero() {
		duration := je.end.Sub(je.start)
		eventext.MergeEventFields(event, mapstr.M{
			"monitor": mapstr.M{
				"duration": mapstr.M{
					"us": duration.Microseconds(),
				},
			},
		})
	}
	eventext.MergeEventFields(event, mapstr.M{
		"url": je.urlFields,
		"event": mapstr.M{
			"type": "heartbeat/summary",
		},
		"synthetics": mapstr.M{
			"type":    "heartbeat/summary",
			"journey": je.journey,
		},
	})

	// Add step count meta for log wrapper
	eventext.SetMeta(event, logger.META_STEP_COUNT, je.stepCount)

	if je.journeyComplete {
		return je.error
	}
	return fmt.Errorf("journey did not finish executing, %d steps ran: %w", je.stepCount, je.error)
}

func stepError(e *SynthError) error {
	return fmt.Errorf("error executing step: %w", e.toECSErr())
}
