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
	"github.com/elastic/go-lookslike"
)

// MonitorTimespanValidator is tests for the `next_run` and `next_run_in.us` keys.
var MonitorTimespanValidator = lookslike.MustCompile(map[string]interface{}{
	"monitor": map[string]interface{}{
		"timespan": map[string]interface{}{
			"gte": IsTime,
			"lt":  IsTime,
		},
	},
})

var MonitorDurationValidator = lookslike.MustCompile(map[string]interface{}{
	"monitor": map[string]interface{}{
		"duration": map[string]interface{}{
			"us": IsInt64,
		},
	},
})
