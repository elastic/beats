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

package fmtstr

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// TimestampFormatString is a wrapper around EventFormatString for the
// common special case where the format expression should only have access to
// shared static fields (typically agent / version) and the event timestamp.
type TimestampFormatString struct {
	eventFormatString *EventFormatString
	fields            common.MapStr
}

// NewTimestampFormatString creates from the given event format string a
// TimestampFormatString that includes only the given static fields and
// a timestamp.
func NewTimestampFormatString(
	eventFormatString *EventFormatString, staticFields common.MapStr,
) (*TimestampFormatString, error) {
	return &TimestampFormatString{
		eventFormatString: eventFormatString,
		fields:            staticFields.Clone(),
	}, nil
}

// FieldsForBeat returns a common.MapStr with the given beat name and
// version assigned to their standard field names.
func FieldsForBeat(beat string, version string) common.MapStr {
	return common.MapStr{
		// beat object was left in for backward compatibility reason for older configs.
		"beat": common.MapStr{
			"name":    beat,
			"version": version,
		},
		"agent": common.MapStr{
			"name":    beat,
			"version": version,
		},
		// For the Beats that have an observer role
		"observer": common.MapStr{
			"name":    beat,
			"version": version,
		},
	}
}

// Run executes the format string returning a new expanded string or an error
// if execution or event field expansion fails.
func (fs *TimestampFormatString) Run(timestamp time.Time) (string, error) {
	placeholderEvent := &beat.Event{
		Fields:    fs.fields,
		Timestamp: timestamp,
	}
	return fs.eventFormatString.Run(placeholderEvent)
}

func (fs *TimestampFormatString) String() string {
	return fs.eventFormatString.expression
}
