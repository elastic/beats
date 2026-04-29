// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"encoding/json"
	"time"
)

// The types in this file mirror the JSON wire format of the Application
// Insights v1 batch metrics endpoint
// (POST https://api.applicationinsights.io/v1/apps/{appId}/metrics).
//
// They replace the equivalent generated types from the deprecated track 1
// SDK package github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights
// so that this module no longer depends on go-autorest.

// MetricsBatchRequestItem is a single entry in the batch metrics request body.
type MetricsBatchRequestItem struct {
	ID         *string                 `json:"id,omitempty"`
	Parameters *MetricsBatchParameters `json:"parameters,omitempty"`
}

// MetricsBatchParameters describes a single metrics query inside a batch request.
type MetricsBatchParameters struct {
	MetricID    string    `json:"metricId"`
	Timespan    *string   `json:"timespan,omitempty"`
	Aggregation *[]string `json:"aggregation,omitempty"`
	Interval    *string   `json:"interval,omitempty"`
	Segment     *[]string `json:"segment,omitempty"`
	Top         *int32    `json:"top,omitempty"`
	Orderby     *string   `json:"orderby,omitempty"`
	Filter      *string   `json:"filter,omitempty"`
}

// ListMetricsResultsItem wraps the JSON array returned by the batch metrics
// endpoint. The wrapper struct (instead of a bare slice) is preserved so that
// callers can still rely on a nil-safe Value pointer, matching the previous
// SDK shape.
type ListMetricsResultsItem struct {
	Value *[]MetricsResultsItem
}

// UnmarshalJSON decodes a top-level JSON array into Value.
func (l *ListMetricsResultsItem) UnmarshalJSON(data []byte) error {
	var arr []MetricsResultsItem
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	l.Value = &arr
	return nil
}

// MarshalJSON encodes Value as a top-level JSON array.
func (l ListMetricsResultsItem) MarshalJSON() ([]byte, error) {
	if l.Value == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(*l.Value)
}

// MetricsResultsItem is a single entry in the batch metrics response.
type MetricsResultsItem struct {
	ID     *string        `json:"id,omitempty"`
	Status *int32         `json:"status,omitempty"`
	Body   *MetricsResult `json:"body,omitempty"`
}

// MetricsResult wraps a MetricsResultInfo as returned by the API.
type MetricsResult struct {
	Value *MetricsResultInfo `json:"value,omitempty"`
}

// MetricsResultInfo contains the actual metric values.
//
// The App Insights v1 metrics API returns a heterogeneous JSON object: a few
// well-known fields (start, end, interval, segments) plus arbitrary metric
// names whose values may be either scalars (for segment values) or nested
// objects keyed by aggregation. Everything that does not match a known field
// is captured into AdditionalProperties so the existing event mapping in
// data.go continues to work unchanged.
type MetricsResultInfo struct {
	Start                *time.Time
	End                  *time.Time
	Interval             *string
	Segments             *[]MetricsSegmentInfo
	AdditionalProperties map[string]interface{}
}

// UnmarshalJSON splits the known fields from the catch-all properties.
func (m *MetricsResultInfo) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["start"]; ok {
		delete(raw, "start")
		if err := json.Unmarshal(v, &m.Start); err != nil {
			return err
		}
	}
	if v, ok := raw["end"]; ok {
		delete(raw, "end")
		if err := json.Unmarshal(v, &m.End); err != nil {
			return err
		}
	}
	if v, ok := raw["interval"]; ok {
		delete(raw, "interval")
		if err := json.Unmarshal(v, &m.Interval); err != nil {
			return err
		}
	}
	if v, ok := raw["segments"]; ok {
		delete(raw, "segments")
		if err := json.Unmarshal(v, &m.Segments); err != nil {
			return err
		}
	}
	if len(raw) == 0 {
		return nil
	}
	m.AdditionalProperties = make(map[string]interface{}, len(raw))
	for k, v := range raw {
		var val interface{}
		if err := json.Unmarshal(v, &val); err != nil {
			return err
		}
		m.AdditionalProperties[k] = val
	}
	return nil
}

// MarshalJSON merges the known fields and AdditionalProperties back into a
// single JSON object. Provided mainly for symmetry and tests; the API itself
// only requires unmarshaling.
func (m MetricsResultInfo) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{}
	for k, v := range m.AdditionalProperties {
		out[k] = v
	}
	if m.Start != nil {
		out["start"] = m.Start
	}
	if m.End != nil {
		out["end"] = m.End
	}
	if m.Interval != nil {
		out["interval"] = m.Interval
	}
	if m.Segments != nil {
		out["segments"] = m.Segments
	}
	return json.Marshal(out)
}

// MetricsSegmentInfo is a node in the segments tree of a metrics response.
//
// Like MetricsResultInfo it has a few known fields plus a catch-all for
// per-metric values and segment dimension labels.
type MetricsSegmentInfo struct {
	Start                *time.Time
	End                  *time.Time
	Segments             *[]MetricsSegmentInfo
	AdditionalProperties map[string]interface{}
}

// UnmarshalJSON splits the known fields from the catch-all properties.
func (m *MetricsSegmentInfo) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["start"]; ok {
		delete(raw, "start")
		if err := json.Unmarshal(v, &m.Start); err != nil {
			return err
		}
	}
	if v, ok := raw["end"]; ok {
		delete(raw, "end")
		if err := json.Unmarshal(v, &m.End); err != nil {
			return err
		}
	}
	if v, ok := raw["segments"]; ok {
		delete(raw, "segments")
		if err := json.Unmarshal(v, &m.Segments); err != nil {
			return err
		}
	}
	if len(raw) == 0 {
		return nil
	}
	m.AdditionalProperties = make(map[string]interface{}, len(raw))
	for k, v := range raw {
		var val interface{}
		if err := json.Unmarshal(v, &val); err != nil {
			return err
		}
		m.AdditionalProperties[k] = val
	}
	return nil
}

// MarshalJSON merges the known fields and AdditionalProperties back into a
// single JSON object.
func (m MetricsSegmentInfo) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{}
	for k, v := range m.AdditionalProperties {
		out[k] = v
	}
	if m.Start != nil {
		out["start"] = m.Start
	}
	if m.End != nil {
		out["end"] = m.End
	}
	if m.Segments != nil {
		out["segments"] = m.Segments
	}
	return json.Marshal(out)
}
