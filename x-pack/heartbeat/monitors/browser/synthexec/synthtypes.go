// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type SynthEvent struct {
	Id                   string        `json:"_id"`
	Type                 string        `json:"type"`
	PackageVersion       string        `json:"package_version"`
	Step                 *Step         `json:"step"`
	Journey              *Journey      `json:"journey"`
	TimestampEpochMicros float64       `json:"@timestamp"`
	Payload              common.MapStr `json:"payload"`
	Blob                 string        `json:"blob"`
	BlobMime             string        `json:"blob_mime"`
	Error                *SynthError   `json:"error"`
	URL                  string        `json:"url"`
	Status               string        `json:"status"`
	RootFields           common.MapStr `json:"root_fields"`
	index                int
}

func (se SynthEvent) ToMap() (m common.MapStr) {
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
		m = common.MapStr{}
	}

	m.DeepUpdate(common.MapStr{
		"synthetics": common.MapStr{
			"type":            se.Type,
			"package_version": se.PackageVersion,
			"index":           se.index,
		},
	})
	if len(se.Payload) > 0 {
		m.Put("synthetics.payload", se.Payload)
	}
	if se.Blob != "" {
		m.Put("synthetics.blob", se.Blob)
	}
	if se.BlobMime != "" {
		m.Put("synthetics.blob_mime", se.BlobMime)
	}
	if se.Step != nil {
		m.Put("synthetics.step", se.Step.ToMap())
	}
	if se.Journey != nil {
		m.Put("synthetics.journey", se.Journey.ToMap())
	}
	if se.Error != nil {
		m.Put("synthetics.error", se.Error.toMap())
	}

	if se.URL != "" {
		u, e := url.Parse(se.URL)
		if e != nil {
			logp.Warn("Could not parse synthetics URL '%s': %s", se.URL, e.Error())
		} else {
			m.Put("url", wrappers.URLFields(u))
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

type SynthError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

func (se *SynthError) String() string {
	return fmt.Sprintf("%s: %s\n", se.Name, se.Message)
}

func (se *SynthError) toMap() common.MapStr {
	return common.MapStr{
		"name":    se.Name,
		"message": se.Message,
		"stack":   se.Stack,
	}
}

type DurationUs struct {
	Micros int64 `json:"us"`
}

func (tu *DurationUs) durationMicros() int64 {
	return tu.Micros
}

func (tu *DurationUs) ToMap() common.MapStr {
	if tu == nil {
		return nil
	}
	return common.MapStr{
		"us": tu.durationMicros(),
	}
}

type Step struct {
	Name     string     `json:"name"`
	Index    int        `json:"index"`
	Status   string     `json:"status"`
	Duration DurationUs `json:"duration"`
}

func (s *Step) ToMap() common.MapStr {
	return common.MapStr{
		"name":     s.Name,
		"index":    s.Index,
		"status":   s.Status,
		"duration": s.Duration.ToMap(),
	}
}

type Journey struct {
	Name string   `json:"name"`
	Id   string   `json:"id"`
	Tags []string `json:"tags"`
}

func (j Journey) ToMap() common.MapStr {
	if len(j.Tags) > 0 {
		return common.MapStr{
			"name": j.Name,
			"id":   j.Id,
			"tags": j.Tags,
		}
	}
	return common.MapStr{
		"name": j.Name,
		"id":   j.Id,
	}
}
