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

package remote_write

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "remote_write",
		remote_write.MetricSetBuilder("prometheus", remoteWriteEventsGeneratorFactory),
		mb.WithHostParser(parse.EmptyHostParser),

		// must replace ensures that we are replacing the oss implementation with this one
		// so we can make use of ES histograms (basic only) when use_types is enabled
		mb.MustReplace(),
	)
}
