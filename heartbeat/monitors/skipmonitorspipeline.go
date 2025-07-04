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
	"github.com/elastic/elastic-agent-libs/logp"
)

type MonitorSkipper interface {
	SkipRunningMonitors(beat.Client) error
}

func WithSkipMonitorPipeline(pipeline beat.Pipeline, skipper MonitorSkipper) beat.Pipeline {
	return &publisherSkipperWrapper{
		pipeline: pipeline,
		skipper:  skipper,
	}
}

type publisherSkipperWrapper struct {
	pipeline beat.Pipeline
	skipper  MonitorSkipper
}

func (hw *publisherSkipperWrapper) ConnectWith(c beat.ClientConfig) (beat.Client, error) {
	return hw.pipeline.ConnectWith(c)
}

func (hw *publisherSkipperWrapper) Connect() (beat.Client, error) {
	return hw.pipeline.Connect()
}

func (hw *publisherSkipperWrapper) Close() error {
	logp.L().Info("=== close pipeline: record skipped monitors ===")

	pubClient, err := hw.Connect()
	if err != nil {
		return err
	}

	if err := hw.skipper.SkipRunningMonitors(pubClient); err != nil {
		return err
	}

	if closer, ok := hw.pipeline.(io.Closer); ok {
		logp.L().Info("=== closing original publisher ===")
		return closer.Close()
	}

	logp.L().Info("=== original publisher is not a closer ===")
	return nil
}
