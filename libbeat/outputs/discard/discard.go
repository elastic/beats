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

package discard

import (
	"context"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	outputs.RegisterType("discard", makeDiscard)
}

type discardOutput struct {
	log      *logp.Logger
	beat     beat.Info
	observer outputs.Observer
}

func makeDiscard(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *config.C,
) (outputs.Group, error) {
	out := &discardOutput{
		log:      logp.NewLogger("discard"),
		beat:     beat,
		observer: observer,
	}
	doConfig := defaultConfig()
	if err := cfg.Unpack(&doConfig); err != nil {
		return outputs.Fail(err)
	}

	// disable bulk support in publisher pipeline
	_ = cfg.SetInt("bulk_max_size", -1, -1)
	out.log.Infof("Initialized discard output")
	return outputs.Success(doConfig.Queue, -1, 0, nil, out)
}

// Implement Outputer
func (out *discardOutput) Close() error {
	return nil
}

func (out *discardOutput) Publish(_ context.Context, batch publisher.Batch) error {
	defer batch.ACK()

	st := out.observer
	events := batch.Events()
	st.NewBatch(len(events))
	st.Acked(len(events))
	return nil
}

func (out *discardOutput) String() string {
	return "discard"
}
