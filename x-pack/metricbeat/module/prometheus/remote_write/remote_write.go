// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	rw "github.com/elastic/beats/v7/metricbeat/module/prometheus/remote_write"
)

func init() {
	// Create base config with size limits from x-pack defaults
	baseConfig := rw.Config{
		Host:                   "localhost",
		Port:                   9201,
		MaxCompressedBodyBytes: defaultConfig.MaxCompressedBodyBytes,
		MaxDecodedBodyBytes:    defaultConfig.MaxDecodedBodyBytes,
	}

	mb.Registry.MustAddMetricSet("prometheus", "remote_write",
		rw.MetricSetBuilderWithConfig(remoteWriteEventsGeneratorFactory, baseConfig),
		mb.WithHostParser(parse.EmptyHostParser),

		// must replace ensures that we are replacing the oss implementation with this one
		// so we can make use of ES histograms (basic only) when use_types is enabled
		mb.MustReplace(),
	)
}
