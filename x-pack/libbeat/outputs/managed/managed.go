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

package pause

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
)

// managed output is used as a placeholder for central management
// this output will cause the beat to pause the output until a real
// output is configured
// It's used both during startup (before we retrieved the current output settings)
// and can be also used to effectively pause the beat
type managed struct{}

func init() {
	outputs.RegisterType("managed", makeManaged)
}

func makeManaged(
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	c := &managed{}
	return outputs.Success(0, 0, c)
}

func (c *managed) Close() error { return nil }
func (c *managed) Publish(batch publisher.Batch) error {
	time.Sleep(60 * time.Second)
	batch.Retry()
	return nil
}

func (c *managed) String() string {
	return "managed"
}
