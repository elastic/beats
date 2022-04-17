// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collector

import (
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/prometheus/collector"
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "collector",
		collector.MetricSetBuilder("prometheus", promEventsGeneratorFactory),
		mb.WithHostParser(collector.HostParser),
		mb.DefaultMetricSet(),

		// must replace ensures that we are replacing the oss implementation with this one
		// so we can make use of ES histograms (basic only) when use_types is enabled
		mb.MustReplace(),
	)
}
