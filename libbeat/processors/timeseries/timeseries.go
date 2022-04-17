// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package timeseries

import (
	"strings"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/cfgwarn"
	"github.com/menderesk/beats/v7/libbeat/mapping"
	"github.com/menderesk/beats/v7/libbeat/processors"

	"github.com/mitchellh/hashstructure"
)

type timeseriesProcessor struct {
	dimensions map[string]interface{}
	prefixes   []string
}

// NewTimeSeriesProcessor returns a processor to add timeseries info to events
// Events are processed to extract all their dimensions (keyword fields that
// hold a dimension of the metrics) and compute a hash of all their values into
// `timeseries.instance` field.
func NewTimeSeriesProcessor(fields mapping.Fields) processors.Processor {
	cfgwarn.Experimental("timeseries.instance field is experimental")

	dimensions := map[string]bool{}
	prefixes := map[string]bool{}
	populateDimensions("", dimensions, prefixes, fields)

	// remove false values and convert to map where a nil value means
	// it's a dimension
	dimensionsNilDict := map[string]interface{}{}
	for k, isDimension := range dimensions {
		if isDimension {
			dimensionsNilDict[k] = nil
		}
	}

	// convert the prefix map to a list
	prefixList := []string{}
	for k, isDimension := range prefixes {
		if isDimension {
			prefixList = append(prefixList, k)
		}
	}

	return &timeseriesProcessor{dimensions: dimensionsNilDict, prefixes: prefixList}
}

func (t *timeseriesProcessor) Run(event *beat.Event) (*beat.Event, error) {
	if event.TimeSeries {
		instanceFields := common.MapStr{}

		// map all dimensions & values
		for k, v := range event.Fields.Flatten() {
			if t.isDimension(k) {
				instanceFields[k] = v
			}
		}

		h, err := hashstructure.Hash(instanceFields, nil)
		if err != nil {
			// this should not happen, keep the event in any case
			return event, err
		}
		event.Fields["timeseries"] = common.MapStr{
			"instance": h,
		}
	}

	return event, nil
}

func (t *timeseriesProcessor) isDimension(field string) bool {
	if _, ok := t.dimensions[field]; ok {
		return true
	}

	// field matches any of the prefixes
	for _, prefix := range t.prefixes {
		if strings.HasPrefix(field, prefix) {
			return true
		}
	}

	return false
}

// put all dimension fields in the given map for quick access
func populateDimensions(prefix string, dimensions map[string]bool, prefixes map[string]bool, fields mapping.Fields) {
	for _, f := range fields {
		name := f.Name
		if prefix != "" {
			name = prefix + "." + name
		}

		if len(f.Fields) > 0 {
			populateDimensions(name, dimensions, prefixes, f.Fields)
			continue
		}

		if f.Type == "object" {
			// everything with this prefix could be a dimension
			name = strings.TrimRight(name, "*")
			if !strings.HasSuffix(name, ".") {
				name += "."
			}
			if _, ok := prefixes[name]; !ok || f.Overwrite {
				prefixes[name] = isDimension(f)
			}
		} else {
			if _, ok := dimensions[name]; !ok || f.Overwrite {
				dimensions[name] = isDimension(f)
			}
		}
	}
}

func isDimension(f mapping.Field) bool {
	// keywords are dimensions by default (disabled with dimension: false in fields.yml)
	if f.Dimension == nil {
		return f.Type == "keyword" || (f.Type == "object" && f.ObjectType == "keyword")
	}

	// user defined dimension (dimension: true in fields.yml)
	return *f.Dimension
}

func (t *timeseriesProcessor) String() string {
	return "timeseries"
}
