// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package synthexec

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type enricher func(event *beat.Event, se *SynthEvent) error

type streamEnricher struct {
	je         *journeyEnricher
	sFields    stdfields.StdMonitorFields
	checkGroup string
}

func newStreamEnricher(sFields stdfields.StdMonitorFields) *streamEnricher {
	return &streamEnricher{sFields: sFields, checkGroup: makeUuid()}
}

func (senr *streamEnricher) enrich(event *beat.Event, se *SynthEvent) error {
	if senr.je == nil || (se != nil && se.Type == JourneyStart) {
		senr.je = newJourneyEnricher(senr)
	}

	return senr.je.enrich(event, se)
}

// journeyEnricher holds state across received SynthEvents retaining fields
// where relevant to properly enrich *beat.Event instances.
type journeyEnricher struct {
	journey        *Journey
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
			je.journey = se.Journey
		case JourneyEnd, CmdStatus:
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
		// noop
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

	return jobErr
}

func stepError(e *SynthError) error {
	return fmt.Errorf("error executing step: %w", e.toECSErr())
}
