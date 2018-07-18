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

package null

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/publisher"
)

func init() {
	outputs.RegisterType("null", makeNullout)
}

type null struct {
	beat     beat.Info
	observer outputs.Observer
	codec    codec.Codec
}

// makeNullout instantiates a new null output instance.
func makeNullout(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	// disable bulk support in publisher pipeline
	cfg.SetInt("bulk_max_size", -1, -1)

	no := &null{
		beat:     beat,
		observer: observer,
	}
	if err := no.init(beat, config); err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(-1, 0, no)
}

func (n *null) init(beat beat.Info, c config) error {
	if c.Codec.Namespace.Name() == "" {
		return nil
	}
	cod, err := codec.CreateEncoder(beat, c.Codec)
	if err != nil {
		return err
	}

	n.codec = cod
	return nil
}

// Implement Outputer
func (n *null) Close() error {
	return nil
}

func (n *null) Publish(
	batch publisher.Batch,
) error {
	defer batch.ACK()

	st := n.observer
	events := batch.Events()
	st.NewBatch(len(events))

	dropped := 0

	if n.codec != nil {
		for i := range events {
			event := &events[i]

			serializedEvent, err := n.codec.Encode(n.beat.Beat, &event.Content)
			if err != nil {
				if event.Guaranteed() {
					logp.Critical("Failed to serialize the event: %v", err)
				} else {
					logp.Warn("Failed to serialize the event: %v", err)
				}

				dropped++
				continue
			}
			st.WriteBytes(len(serializedEvent) + 1)
		}
	}

	st.Dropped(dropped)
	st.Acked(len(events) - dropped)

	return nil
}
