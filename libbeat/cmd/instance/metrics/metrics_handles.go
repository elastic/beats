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
// +build windows

package metrics

import (
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/elastic-agent-libs/logp"
	sysinfo "github.com/elastic/go-sysinfo"
	"github.com/elastic/go-sysinfo/types"
)

const (
	fileHandlesNotReported = "Following metrics will not be reported: beat.handles.open"
)

var (
	handleCounter types.OpenHandleCounter
)

func setupWindowsHandlesMetrics() {
	beatProcessSysInfo, err := sysinfo.Self()
	if err != nil {
		logp.Err("Error while getting own process info: %v", err)
		logp.Err(fileHandlesNotReported)
		return
	}

	var ok bool
	handleCounter, ok = beatProcessSysInfo.(types.OpenHandleCounter)
	if !ok {
		logp.Err("Process does not implement types.OpenHandleCounter: %v", beatProcessSysInfo)
		logp.Err(fileHandlesNotReported)
		return
	}

	monitoring.NewFunc(beatMetrics, "handles", reportOpenHandles, monitoring.Report)
}

func reportOpenHandles(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	n, err := handleCounter.OpenHandleCount()
	if err != nil {
		logp.Err("Error while retrieving the number of open file handles: %v", err)
		return
	}

	monitoring.ReportInt(V, "open", int64(n))
}
