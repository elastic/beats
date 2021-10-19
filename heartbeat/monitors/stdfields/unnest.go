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

package stdfields

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

// OptionalStream represents a config that has a stream set, which in practice
// means agent/fleet. In this case we primarily use the first stream for the
// config, but we do pull the Id from the root, and merge the root data stream
// in as well
type OptionalStream struct {
	Id         string           `config:"id"`
	DataStream *common.Config   `config:"data_stream"`
	Streams    []*common.Config `config:"streams"`
}

type BaseStream struct {
	Type string `config:"type"`
}

// UnnestStream detects configs that come from fleet and transforms the config into something compatible
// with heartbeat, by mixing some fields (id, data_stream) with those from the first stream. It assumes
// that there is exactly one stream associated with the input.
func UnnestStream(config *common.Config) (res *common.Config, err error) {
	optS := &OptionalStream{}
	err = config.Unpack(optS)
	if err != nil {
		return nil, fmt.Errorf("could not unnest stream: %w", err)
	}

	if len(optS.Streams) == 0 {
		return config, nil
	}

	// Find the 'base' stream, that is the one stream that has `type` set.
	// The other streams are sort of ancillary and only for fleet internals, the
	// base stream has the full monitor config contained within
	for _, stream := range optS.Streams {
		bs := &BaseStream{}
		stream.Unpack(bs)
		if bs.Type != "" {
			res = stream
			break
		}
	}

	if res == nil {
		id, _ := config.String("id", 0)
		return nil, fmt.Errorf("could not determine base stream for config: %s", id)
	}

	err = res.Merge(common.MapStr{"id": optS.Id, "data_stream": optS.DataStream})
	return
}
