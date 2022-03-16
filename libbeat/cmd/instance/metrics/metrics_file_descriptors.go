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

//go:build linux || (freebsd && cgo)
// +build linux freebsd,cgo

package metrics

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
)

func setupLinuxBSDFDMetrics() {
	monitoring.NewFunc(beatMetrics, "handles", reportFDUsage, monitoring.Report)
}

func reportFDUsage(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	open, hardLimit, softLimit, err := getFDUsage()
	if err != nil {
		logp.Err("Error while retrieving FD information: %v", err)
		return
	}

	monitoring.ReportInt(V, "open", int64(open))
	monitoring.ReportNamespace(V, "limit", func() {
		monitoring.ReportInt(V, "hard", int64(hardLimit))
		monitoring.ReportInt(V, "soft", int64(softLimit))
	})
}

func getFDUsage() (open, hardLimit, softLimit uint64, err error) {

	state, err := beatProcessStats.GetSelf()
	if err != nil {
		return 0, 0, 0, errors.Wrap(err, "error fetching self process")
	}

	open = state.FD.Open.ValueOr(0)

	hardLimit = state.FD.Limit.Hard.ValueOr(0)

	softLimit = state.FD.Limit.Soft.ValueOr(0)

	return open, hardLimit, softLimit, nil
}
