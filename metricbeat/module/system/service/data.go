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

//go:build linux
// +build linux

package service

import (
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Properties is a struct representation of the dbus returns from GetAllProperties
type Properties struct {
	ExecMainCode   int32
	ExecMainStatus int32
	ExecMainPID    uint32
	// accounting
	CPUAccounting    bool
	MemoryAccounting bool
	TasksAccounting  bool
	IPAccounting     bool
	// metrics
	CPUUsageNSec     int64
	MemoryCurrent    int64
	TasksCurrent     int64
	IPIngressPackets int64
	IPIngressBytes   int64
	IPEgressPackets  int64
	IPEgressBytes    int64
	// timestamps
	ActiveEnterTimestamp   uint64
	InactiveEnterTimestamp uint64
	InactiveExitTimestamp  uint64
	ActiveExitTimestamp    uint64
	// Meta
	FragmentPath string
	// UnitFileState
	UnitFileState  string
	UnitFilePreset string
}

// formProperties gets properties for the systemd service and returns a MapStr with useful data
func formProperties(unit dbus.UnitStatus, props Properties) (mb.Event, error) {
	timeSince, err := timeSince(props, unit.ActiveState)
	if err != nil {
		return mb.Event{}, errors.Wrap(err, "error getting timestamp")
	}

	event := mb.Event{
		RootFields: mapstr.M{},
	}
	msData := mapstr.M{
		"name":       unit.Name,
		"load_state": unit.LoadState,
		"state":      unit.ActiveState,
		"sub_state":  unit.SubState,
		"unit_file": mapstr.M{
			"state":         props.UnitFileState,
			"vendor_preset": props.UnitFilePreset,
		},
	}

	//most of the properties values are context-dependent.
	//If things aren't running/haven't run/etc than a lot of the values should be ignored.

	//Even systemd doesn't check the substate, leading to a lot of odd `Memory: 0B` lines in `systemctl status`
	//Ignore the resource accounting if a service has exited
	if unit.ActiveState == "active" && unit.SubState != "exited" {
		msData["resources"] = getMetricsFromServivce(props)
	}

	var childProc = mapstr.M{}
	childData := false
	//anything less than 1 isn't a valid SIGCHLD code
	if props.ExecMainCode > 0 {
		childData = true
		msData["exec_code"] = translateChild(props.ExecMainCode)
		childProc["exit_code"] = props.ExecMainStatus
	}

	//only send timestamp if it's valid
	if timeSince != 0 {
		msData["state_since"] = time.Unix(0, timeSince)
	}

	//only prints PID data if we have a PID
	if props.ExecMainPID > 0 {
		childData = true
		childProc["pid"] = props.ExecMainPID
	}

	if childData {
		event.RootFields["process"] = childProc
	}
	if props.IPAccounting {
		event.RootFields["network"] = mapstr.M{
			"packets": props.IPIngressPackets + props.IPEgressPackets,
			"bytes":   props.IPIngressBytes + props.IPEgressBytes,
		}
	}
	event.RootFields["systemd"] = mapstr.M{
		"unit":          unit.Name,
		"fragment_path": props.FragmentPath,
	}

	event.MetricSetFields = msData

	return event, nil
}

// getMetricsFromServivce checks what accounting we have enabled and uses that to determine what metrics we can send back to the user
func getMetricsFromServivce(props Properties) mapstr.M {
	metrics := mapstr.M{}

	if props.CPUAccounting {
		metrics["cpu"] = mapstr.M{
			"usage": mapstr.M{
				"nsec": props.CPUUsageNSec,
			},
		}
	}

	if props.MemoryAccounting {
		metrics["memory"] = mapstr.M{
			"usage": mapstr.M{
				"bytes": props.MemoryCurrent,
			},
		}
	}

	if props.TasksAccounting {
		metrics["tasks"] = mapstr.M{
			"count": props.TasksCurrent,
		}
	}

	if props.IPAccounting {
		metrics["network"] = mapstr.M{
			"in": mapstr.M{
				"packets": props.IPIngressPackets,
				"bytes":   props.IPIngressBytes,
			},
			"out": mapstr.M{
				"packets": props.IPEgressPackets,
				"bytes":   props.IPEgressBytes,
			},
		}
	}

	return metrics
}

// translateChild translates the SIGCHILD code that systemd gets from the MainPID under its control into a string value.
// Normally this shows up in systemctl status like this:  Main PID: 5305 (code=exited, status=0/SUCCESS)
// This mapping of SIGCHILD int codes comes from the kernel. systemd does something similar to turn them into pretty strings
func translateChild(code int32) string {
	/*
	   CLD_EXITED: 1
	   CLD_KILLED: 2
	   CLD_DUMPED: 3
	   CLD_TRAPPED: 4
	   CLD_STOPPED: 5
	   CLD_CONTINUED: 6
	*/
	switch code {
	case 1:
		return "exited"
	case 2:
		return "killed"
	case 3:
		return "dumped"
	case 4:
		return "trapped"
	case 5:
		return "stopped"
	case 6:
		return "continued"
	default:
		return "unknown"
	}

}

// timeSince emulates the behavior of `systemctl status` with regards to reporting time:
// "Active: inactive (dead) since Sat 2019-10-19 16:31:25 PDT;""
// The dbus properties data contains a number of different timestamps, which one systemd reports depends on the unit state
func timeSince(props Properties, state string) (int64, error) {

	var ts uint64
	//normally we would want to check these values before we cast,
	//but checking the state before we access `props` insures the values will be there.
	switch state {
	case "reloading", "active":
		ts = props.ActiveEnterTimestamp
	case "failed", "inactive":
		ts = props.InactiveEnterTimestamp
	case "activating":
		ts = props.InactiveExitTimestamp
	default:
		ts = props.ActiveExitTimestamp
	}

	//That second number is "USEC_INFINITY" which seems to a thing systemd cares about
	if ts <= 0 || ts == 18446744073709551615 {
		return 0, nil
	}

	//convert from usec
	return int64(ts) * 1000, nil

}
