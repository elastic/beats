// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stackdriver

import (
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/genproto/googleapis/monitoring/v3"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func newIncomingFieldExtractor(l *logp.Logger) *incomingFieldExtractor {
	return &incomingFieldExtractor{logger: l}
}

type incomingFieldExtractor struct {
	logger *logp.Logger
}

// KeyValuePoint is a struct to capture the information parsed in an instant of a single metric
type KeyValuePoint struct {
	Key       string
	Value     interface{}
	Labels    common.MapStr
	ECS       common.MapStr
	Timestamp time.Time
}

// extractTimeSeriesMetricValues valuable to send to Elasticsearch. This includes, for example, metric values, labels and timestamps
func (e *incomingFieldExtractor) extractTimeSeriesMetricValues(resp *monitoring.TimeSeries) (points []KeyValuePoint, err error) {
	points = make([]KeyValuePoint, 0)

	for _, point := range resp.Points {
		// Don't add point intervals that can't be "stated" at some timestamp.
		ts, err := e.getTimestamp(point)
		if err != nil {
			e.logger.Warn(err)
			continue
		}

		p := KeyValuePoint{
			Key:       cleanMetricNameString(resp.Metric.Type),
			Value:     getValueFromPoint(point),
			Timestamp: ts,
		}

		points = append(points, p)
	}

	return points, nil
}

func (e *incomingFieldExtractor) getTimestamp(p *monitoring.Point) (ts time.Time, err error) {
	// Don't add point intervals that can't be "stated" at some timestamp.
	if p.Interval != nil {
		if ts, err = ptypes.Timestamp(p.Interval.StartTime); err != nil {
			return time.Time{}, errors.Errorf("error trying to parse timestamp '%#v' from metric\n", p.Interval.StartTime)
		}
		return ts, nil
	}

	return time.Time{}, errors.New("error trying to extract the timestamp from the point data")
}

var rx = regexp.MustCompile(`^[a-z_-]+\.googleapis.com\/`)

func cleanMetricNameString(s string) string {
	if s == "" {
		return "unknown"
	}

	prefix := rx.FindString(s)

	removedPrefix := strings.TrimPrefix(s, prefix)
	replacedChars := strings.Replace(removedPrefix, "/", ".", -1)

	return replacedChars
}

func getValueFromPoint(p *monitoring.Point) (out interface{}) {
	switch v := p.Value.Value.(type) {
	case *monitoring.TypedValue_DoubleValue:
		out = v.DoubleValue
	case *monitoring.TypedValue_BoolValue:
		out = v.BoolValue
	case *monitoring.TypedValue_Int64Value:
		out = v.Int64Value
	case *monitoring.TypedValue_StringValue:
		out = v.StringValue
	case *monitoring.TypedValue_DistributionValue:
		//TODO Distribution values aren't simple values. Take a look at this
		out = v.DistributionValue
	}

	return out
}
