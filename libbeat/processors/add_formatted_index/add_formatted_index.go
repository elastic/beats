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

package add_formatted_index

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// AddFormattedIndex is a Processor to set an event's "raw_index" metadata field
// with a given TimestampFormatString. The elasticsearch output interprets
// that field as specifying the (raw string) index the event should be sent to;
// in other outputs it is just included in the metadata.
type AddFormattedIndex struct {
	formatString *fmtstr.TimestampFormatString
}

// New returns a new AddFormattedIndex processor.
func New(formatString *fmtstr.TimestampFormatString) *AddFormattedIndex {
	return &AddFormattedIndex{formatString}
}

// Run runs the processor.
func (p *AddFormattedIndex) Run(event *beat.Event) (*beat.Event, error) {
	index, err := p.formatString.Run(event.Timestamp)
	if err != nil {
		return nil, err
	}

	if event.Meta == nil {
		event.Meta = mapstr.M{}
	}
	event.Meta[events.FieldMetaRawIndex] = index
	return event, nil
}

func (p *AddFormattedIndex) String() string {
	return fmt.Sprintf("add_index_pattern=%v", p.formatString)
}
