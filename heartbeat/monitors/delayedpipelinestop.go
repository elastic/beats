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

package monitors

import (
	"io"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func WithDelayedPipelineStop(pipeline beat.Pipeline, closeCh chan struct{}) beat.Pipeline {
	p := &delayedPipelineStop{
		pipeline: pipeline,
	}

	go func() {
		<-closeCh
		p.closePipeline()
	}()
	return p
}

type delayedPipelineStop struct {
	pipeline beat.Pipeline
}

func (ps *delayedPipelineStop) ConnectWith(c beat.ClientConfig) (beat.Client, error) {
	return ps.pipeline.ConnectWith(c)
}

func (ps *delayedPipelineStop) Connect() (beat.Client, error) {
	return ps.pipeline.Connect()
}

func (ps *delayedPipelineStop) Close() error {
	// Do not close the underlying pipeline immediately.
	// Manually close it by listening to the channel in the constructor.
	return nil
}

func (ps *delayedPipelineStop) closePipeline() error {
	if closer, ok := ps.pipeline.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
