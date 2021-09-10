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

package inputsource

import (
	"io"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/reader"
)

type NetworkMessageReader struct {
	ready    bool
	raw      []byte
	metadata NetworkMetadata
}

func (r *NetworkMessageReader) SetData(raw []byte, metadata NetworkMetadata) {
	r.raw = raw
	r.metadata = metadata
	r.ready = true
}

func (p *NetworkMessageReader) Next() (reader.Message, error) {
	if !p.ready {
		return reader.Message{}, io.EOF
	}
	p.ready = false

	fields := common.MapStr{}
	return reader.Message{
		Content: p.raw,
		Bytes:   len(p.raw),
		Fields:  fields,
	}, nil
}

func (p *NetworkMessageReader) Close() error {
	p.ready = false
	return nil
}
