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

package report

import (
	"context"

	"github.com/shirou/gopsutil/v4/common"
	psprocess "github.com/shirou/gopsutil/v4/process"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
)

func SetupLinuxBSDFDMetrics(logger *logp.Logger, reg *monitoring.Registry, processStats *process.Stats) {
	monitoring.NewFunc(reg, "handles", FDUsageReporter(logger, processStats), monitoring.Report)
}

func FDUsageReporter(logger *logp.Logger, processStats *process.Stats) func(_ monitoring.Mode, V monitoring.Visitor) {
	pid, err := process.GetSelfPid(processStats.Hostfs)
	if err != nil {
		logger.Error("Error while retrieving pid: %v", err)
		return func(_ monitoring.Mode, V monitoring.Visitor) {
			V.OnRegistryStart()
			V.OnRegistryFinished()
		}
	}
	p := psprocess.Process{
		Pid: int32(pid),
	}

	ctx := context.Background()
	if processStats != nil && processStats.Hostfs != nil && processStats.Hostfs.IsSet() {
		ctx = context.WithValue(context.Background(), common.EnvKey, common.EnvMap{common.HostProcEnvKey: processStats.Hostfs.ResolveHostFS("/proc")})
	}

	return func(_ monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		open, err := p.NumFDsWithContext(ctx)
		if err != nil {
			logger.Errorf("Error while retrieving open FDs information: %v", err)
			return
		}

		stats, err := p.RlimitWithContext(ctx)
		if err != nil {
			logger.Errorf("Error while retrieving FD stats information: %v", err)
			return
		}

		hardLimit := 0
		softLimit := 0
		for _, stat := range stats {
			if stat.Resource == psprocess.RLIMIT_NOFILE {
				hardLimit = int(stat.Hard)
				softLimit = int(stat.Soft)
			}
		}

		monitoring.ReportInt(V, "open", int64(open))
		monitoring.ReportNamespace(V, "limit", func() {
			monitoring.ReportInt(V, "hard", int64(hardLimit))
			monitoring.ReportInt(V, "soft", int64(softLimit))
		})
	}
}
