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

package cursor

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/unison"
)

// cleaner removes finished entries from the registry file.
type cleaner struct {
	log *logp.Logger
}

// run starts a loop that tries to clean entries from the registry.
// The cleaner locks the store, such that no new states can be created
// during the cleanup phase. Only resources that are finished and whos TTL
// (clean_timeout setting) has expired will be removed.
//
// Resources are considered "Finished" if they do not have a current owner (active input), and
// if they have no pending updates that still need to be written to the registry file after associated
// events have been ACKed by the outputs.
// The event acquisition timestamp is used as reference to clean resources. If a resources was blocked
// for a long time, and the life time has been exhausted, then the resource will be removed immediately
// once the last event has been ACKed.
func (c *cleaner) run(canceler unison.Canceler, store *store, interval time.Duration) {
	panic("TODO: implement me")
}
