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

package report

import (
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
)

func SetupLinuxBSDFDMetrics(logger *logp.Logger, reg *monitoring.Registry, processStats *process.Stats) {
	monitoring.NewFunc(reg, "handles", FDUsageReporter(logger, processStats), monitoring.Report)
}

func FDUsageReporter(logger *logp.Logger, processStats *process.Stats) func(_ monitoring.Mode, V monitoring.Visitor) {
	return func(_ monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		open, hardLimit, softLimit, err := getFDUsage(processStats)
		if err != nil {
			logger.Error("Error while retrieving FD information: %v", err)
			return
		}

		monitoring.ReportInt(V, "open", int64(open))
		monitoring.ReportNamespace(V, "limit", func() {
			monitoring.ReportInt(V, "hard", int64(hardLimit))
			monitoring.ReportInt(V, "soft", int64(softLimit))
		})
	}
}

func getFDUsage(processStats *process.Stats) (open, hardLimit, softLimit uint64, err error) {
	state, err := processStats.GetSelf()
	if err != nil {
		return 0, 0, 0, err
	}

	return state.FD.Open.ValueOr(0), state.FD.Limit.Hard.ValueOr(0), state.FD.Limit.Soft.ValueOr(0), nil
}
