// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
	"github.com/elastic/beats/v8/metricbeat/module/prometheus/remote_write"
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "remote_write",
		remote_write.MetricSetBuilder(remoteWriteEventsGeneratorFactory),
		mb.WithHostParser(parse.EmptyHostParser),

		// must replace ensures that we are replacing the oss implementation with this one
		// so we can make use of ES histograms (basic only) when use_types is enabled
		mb.MustReplace(),
	)
}
