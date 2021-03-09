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

package hbtestllext

import (
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

// MonitorTimespanValidator is tests for the `next_run` and `next_run_in.us` keys.
var MonitorTimespanValidator = lookslike.MustCompile(map[string]interface{}{
	"monitor": map[string]interface{}{
		"timespan": IsTimespanBounds,
	},
})

// IsTimespanBounds checks whether the given value is a schedule.TimespanBounds object
var IsTimespanBounds = isdef.Is("a timespan", func(path llpath.Path, v interface{}) *llresult.Results {
	_, ok := v.(schedule.TimespanBounds)
	return llresult.SimpleResult(path, ok, "expected a TimespanBounds struct, got %#v", v)
})
