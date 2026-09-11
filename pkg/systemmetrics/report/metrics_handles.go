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

//go:build windows

package report

import (
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	sysinfo "github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	fileHandlesNotReported = "Following metrics will not be reported: beat.handles.open"
)

var (
	handleCounter types.OpenHandleCounter
)

func SetupWindowsHandlesMetrics(logger *logp.Logger, reg *monitoring.Registry) {
	beatProcessSysInfo, err := sysinfo.Self()
	if err != nil {
		logger.Error("Error while getting own process info: %v", err)
		logger.Error(fileHandlesNotReported)
		return
	}

	var ok bool
	handleCounter, ok = beatProcessSysInfo.(types.OpenHandleCounter)
	if !ok {
		logger.Error("Process does not implement types.OpenHandleCounter: %v", beatProcessSysInfo)
		logger.Error(fileHandlesNotReported)
		return
	}

	monitoring.NewFunc(reg, "handles", openHandlesReporter(logger), monitoring.Report)
}

func openHandlesReporter(logger *logp.Logger) func(_ monitoring.Mode, V monitoring.Visitor) {
	return func(_ monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		n, err := handleCounter.OpenHandleCount()
		if err != nil {
			logger.Error("Error while retrieving the number of open file handles: %v", err)
			return
		}
		monitoring.ReportInt(V, "open", int64(n))
	}
}
