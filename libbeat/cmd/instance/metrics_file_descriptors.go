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

// +build linux freebsd,cgo

package instance

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
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
	state, err := getBeatProcessState()
	if err != nil {
		return 0, 0, 0, err
	}

	iOpen, err := state.GetValue("fd.open")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error getting number of open FD: %v", err)
	}

	open, ok := iOpen.(uint64)
	if !ok {
		return 0, 0, 0, fmt.Errorf("error converting value of open FDs to uint64: %v", iOpen)
	}

	iHardLimit, err := state.GetValue("fd.limit.hard")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error getting FD hard limit: %v", err)
	}

	hardLimit, ok = iHardLimit.(uint64)
	if !ok {
		return 0, 0, 0, fmt.Errorf("error converting values of FD hard limit: %v", iHardLimit)
	}

	iSoftLimit, err := state.GetValue("fd.limit.soft")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("error getting FD hard limit: %v", err)
	}

	softLimit, ok = iSoftLimit.(uint64)
	if !ok {
		return 0, 0, 0, fmt.Errorf("error converting values of FD hard limit: %v", iSoftLimit)
	}

	return open, hardLimit, softLimit, nil
}
