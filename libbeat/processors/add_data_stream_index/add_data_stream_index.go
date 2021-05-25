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

package add_data_stream_index

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
)

const FieldMetaCustomDataset = "dataset"

func SetEventDataset(event *beat.Event, ds string) {
	if event.Meta == nil {
		event.Meta = common.MapStr{
			FieldMetaCustomDataset: ds,
		}
	} else {
		event.Meta[FieldMetaCustomDataset] = ds
	}
}

// AddDataStreamIndex is a Processor to set an event's "raw_index" metadata field
// based on the given type, dataset, and namespace fields.
// If the event's metadata contains an
type AddDataStreamIndex struct {
	DataStream DataStream
	// cached, compiled version of the index name derived from the data stream
	dsCached      string
	customDsCache string
}

// New returns a new AddDataStreamIndex processor.
func New(ds DataStream) *AddDataStreamIndex {
	if ds.Namespace == "" {
		ds.Namespace = "default"
	}
	if ds.Dataset == "" {
		ds.Dataset = "generic"
	}
	return &AddDataStreamIndex{
		DataStream:    ds,
		dsCached:      ds.indexName(),
		customDsCache: ds.datasetFmtString(),
	}
}

// Run runs the processor.
func (p *AddDataStreamIndex) Run(event *beat.Event) (*beat.Event, error) {
	if event.Meta == nil {
		event.Meta = common.MapStr{
			events.FieldMetaRawIndex: p.dsCached,
		}
	} else {
		customDs, hasCustom := event.Meta[FieldMetaCustomDataset]
		if !hasCustom {
			event.Meta[events.FieldMetaRawIndex] = p.dsCached
		} else {
			event.Meta[events.FieldMetaRawIndex] = fmt.Sprintf(p.customDsCache, customDs)
		}
	}

	return event, nil
}

func (p *AddDataStreamIndex) String() string {
	return fmt.Sprintf("add_data_stream_index=%v", p.DataStream.indexName())
}

// DataStream represents the 3-tuple + configuration metadata since it
// can be convenient to import this into other contexts.
type DataStream struct {
	Namespace string `config:"namespace"`
	Dataset   string `config:"dataset"`
	Type      string `config:"type"`
}

func (ds DataStream) datasetFmtString() string {
	return fmt.Sprintf("%s-%%s-%s", ds.Type, ds.Namespace)
}

func (ds DataStream) indexName() string {
	return fmt.Sprintf(ds.datasetFmtString(), ds.Dataset)
}
