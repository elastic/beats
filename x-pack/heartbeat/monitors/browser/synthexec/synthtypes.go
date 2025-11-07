// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package synthexec

import (
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/wraputil"
)

// These constants define all known synthetics event types
const (
	JourneyStart       = "journey/start"
	JourneyEnd         = "journey/end"
	JourneyNetworkInfo = "journey/network_info"
	StepStart          = "step/start"
	StepEnd            = "step/end"
	CmdStatus          = "cmd/status"
	StepScreenshot     = "step/screenshot"
	StepScreenshotRef  = "step/screenshot_ref"
	ScreenshotBlock    = "screenshot/block"
	Stdout             = "stdout" // special type for non-JSON content read off stdout
	Stderr             = "stderr" // special type for content read off stderr
)

type SynthEvent struct {
	Id                   string      `json:"_id"`
	Type                 string      `json:"type"`
	PackageVersion       string      `json:"package_version"`
	Step                 *Step       `json:"step"`
	Journey              *Journey    `json:"journey"`
	TimestampEpochMicros float64     `json:"@timestamp"`
	Payload              mapstr.M    `json:"payload"`
	Blob                 string      `json:"blob"`
	BlobMime             string      `json:"blob_mime"`
	Error                *SynthError `json:"error"`
	URL                  string      `json:"url"`
	Status               string      `json:"status"`
	RootFields           mapstr.M    `json:"root_fields"`
	index                int
}

func (se SynthEvent) ToMap() (m mapstr.M) {
	// We don't add @timestamp to the map string since that's specially handled in beat.Event
	// Use the root fields as a base, and layer additional, stricter, fields on top
	if se.RootFields != nil {
		m = se.RootFields
		// We handle url specially since it can be passed as a string,
		// but expanded to match ECS
		if urlStr, ok := m["url"].(string); ok {
			if se.URL == "" {
				se.URL = urlStr
			}
		}
	} else {
		m = mapstr.M{}
	}

	m.DeepUpdate(mapstr.M{
		"synthetics": mapstr.M{
			"type":            se.Type,
			"package_version": se.PackageVersion,
			"index":           se.index,
		},
	})
	if len(se.Payload) > 0 {
		_, _ = m.Put("synthetics.payload", se.Payload)
	}
	if se.Blob != "" {
		_, _ = m.Put("synthetics.blob", se.Blob)
	}
	if se.BlobMime != "" {
		_, _ = m.Put("synthetics.blob_mime", se.BlobMime)
	}
	if se.Step != nil {
		_, _ = m.Put("synthetics.step", se.Step.ToMap())
	}
	if se.Journey != nil {
		_, _ = m.Put("synthetics.journey", se.Journey.ToMap())
	}
	if se.Error != nil {
		_, _ = m.Put("synthetics.error", se.Error.toMap())
	}

	if se.URL != "" {
		u, e := url.Parse(se.URL)
		if e != nil {
			_, _ = m.Put("url", mapstr.M{"full": se.URL})
			logp.L().Warnf("Could not parse synthetics URL '%s': %s", se.URL, e.Error())
		} else {
			_, _ = m.Put("url", wraputil.URLFields(u))
		}
	}

	return m
}

func (se SynthEvent) Timestamp() time.Time {
	seconds := se.TimestampEpochMicros / 1e6
	wholeSeconds := math.Floor(seconds)
	micros := (seconds - wholeSeconds) * 1e6
	nanos := micros * 1000
	return time.Unix(int64(wholeSeconds), int64(nanos))
}

// SynthError describes an error coming out of the synthetics agent
// At some point we should deprecate this in favor of ECSErr and unify the behavior
// to just follow ECS schema everywhere.
type SynthError struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Stack   string `json:"stack"`
}

func (se *SynthError) String() string {
	return fmt.Sprintf("%s: %s\n", se.Name, se.Message)
}

func (se *SynthError) toMap() mapstr.M {
	return mapstr.M{
		"name":    se.Name,
		"message": se.Message,
		"stack":   se.Stack,
	}
}

func (se *SynthError) toECSErr() *ecserr.ECSErr {
	// Type is more ECS friendly, so we prefer it
	t := se.Type
	if t == "" {
		// Legacy support for the 'name' field
		t = se.Name

	}

	var stack *string
	if se.Stack != "" {
		stack = &se.Stack
	}
	return ecserr.NewECSErrWithStack(
		ecserr.EType(t),
		ecserr.ECode(se.Code),
		se.Message,
		stack,
	)
}

// ECSErrToSynthError does a simple type conversion. Hopefully at
// some point we can move away from SynthError.
func ECSErrToSynthError(ee *ecserr.ECSErr) *SynthError {
	var stack string
	if ee.StackTrace != nil {
		stack = *ee.StackTrace
	}
	return &SynthError{
		Type:    string(ee.Type),
		Code:    string(ee.Code),
		Message: ee.Message,
		Stack:   stack,
	}
}

func (se *SynthError) Error() string {
	return se.toECSErr().Error()
}

type DurationUs struct {
	Micros int64 `json:"us"`
}

func (tu *DurationUs) durationMicros() int64 {
	return tu.Micros
}

func (tu *DurationUs) ToMap() mapstr.M {
	if tu == nil {
		return nil
	}
	return mapstr.M{
		"us": tu.durationMicros(),
	}
}

type Step struct {
	Name     string     `json:"name"`
	Index    int        `json:"index"`
	Status   string     `json:"status"`
	Duration DurationUs `json:"duration"`
}

func (s *Step) ToMap() mapstr.M {
	return mapstr.M{
		"name":     s.Name,
		"index":    s.Index,
		"status":   s.Status,
		"duration": s.Duration.ToMap(),
	}
}

type Journey struct {
	Name string   `json:"name"`
	ID   string   `json:"id"`
	Tags []string `json:"tags"`
}

func (j Journey) ToMap() mapstr.M {
	if len(j.Tags) > 0 {
		return mapstr.M{
			"name": j.Name,
			"id":   j.ID,
			"tags": j.Tags,
		}
	}
	return mapstr.M{
		"name": j.Name,
		"id":   j.ID,
	}
}
