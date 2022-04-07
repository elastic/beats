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

package mock

import (
	"time"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/cmd/instance"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

///*** Mock Beat Setup ***///

var Version = "9.9.9"
var Name = "mockbeat"

var Settings = instance.Settings{Name: Name, Version: Version, HasDashboards: true}

type Mockbeat struct {
	done   chan struct{}
	logger *logp.Logger
}

// Creates beater
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	return &Mockbeat{
		done:   make(chan struct{}),
		logger: logp.NewLogger("mock"),
	}, nil
}

/// *** Beater interface methods ***///

func (mb *Mockbeat) Run(b *beat.Beat) error {
	client, err := b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				client.Publish(beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						"type":    "mock",
						"message": "Mockbeat is alive!",
					},
				})
			case <-mb.done:
				ticker.Stop()
				return
			}
		}
	}()

	<-mb.done
	return nil
}

func (mb *Mockbeat) Stop() {
	mb.logger.Info("Mockbeat Stop")

	close(mb.done)
}
