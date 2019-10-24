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

package services

import (
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// Properties is a struct representation of the dbus returns from GetAllProperties
type Properties struct {
	ExecMainCode   int32
	ExecMainStatus int32
	ExecMainPID    uint32
	//Accounting
	CPUAccounting          bool
	MemoryAccounting       bool
	TasksAccounting        bool
	IPAccounting           bool
	CPUUsageNSec           int64
	MemoryCurrent          int64
	TasksCurrent           int64
	IPIngressPackets       int64
	IPIngressBytes         int64
	IPEgressPackets        int64
	IPEgressBytes          int64
	ActiveEnterTimestamp   uint64
	InactiveEnterTimestamp uint64
	InactiveExitTimestamp  uint64
	ActiveExitTimestamp    uint64
}

// formProperties gets properties for the systemd service and returns a MapStr with useful data
func formProperties(unit dbus.UnitStatus, props Properties) (common.MapStr, error) {
	timeSince, err := timeSince(props, unit.ActiveState)
	if err != nil {
		return nil, errors.Wrap(err, "error getting timestamp")
	}

	event := common.MapStr{
		"name":         unit.Name,
		"load_state":   unit.LoadState,
		"active_state": unit.ActiveState,
		"sub_state":    unit.SubState,
	}

	//most of the properties values are context-dependent.
	//If things aren't running/haven't run/etc than a lot of the values should be ignored.

	//Even systemd doesn't check the substate, leading to a lot of odd `Memory: 0B` lines in `systemctl status`
	//Ignore the resource accounting if a service has exited
	if unit.ActiveState == "active" && unit.SubState != "exited" {
		event["resources"] = getMetricsFromServivce(props)

	}

	//anything less than 1 isn't a valid SIGCHLD code
	if props.ExecMainCode > 0 {
		event["exec_code"] = translateChild(props.ExecMainCode)
		event["exec_rc"] = props.ExecMainStatus
	}

	//only send timestamp if it's valid
	if timeSince != 0 {
		event["state_since"] = time.Unix(0, timeSince)
	}

	//only prints PID data if we have a PID
	if props.ExecMainPID > 0 {
		event["main_pid"] = props.ExecMainPID
	}

	return event, nil
}

// getMetricsFromServivce checks what accounting we have enabled and uses that to determine what metrics we can send back to the user
func getMetricsFromServivce(props Properties) common.MapStr {
	metrics := common.MapStr{}
	//This error checking is because we don't _quite_ trust the maps we get back from the API.
	if props.CPUAccounting {
		metrics["cpu"] = common.MapStr{
			"usage": common.MapStr{
				"nsec": props.CPUUsageNSec,
			},
		}
	}

	if props.MemoryAccounting {
		metrics["memory"] = common.MapStr{
			"usage": common.MapStr{
				"bytes": props.MemoryCurrent,
			},
		}
	}

	if props.TasksAccounting {
		metrics["tasks"] = common.MapStr{
			"count": props.TasksCurrent,
		}
	}

	if props.IPAccounting {
		metrics["network"] = common.MapStr{
			"in": common.MapStr{
				"packets": props.IPIngressPackets,
				"bytes":   props.IPIngressBytes,
			},
			"out": common.MapStr{
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
	if state == "active" || state == "reloading" {
		ts = props.ActiveEnterTimestamp
	} else if state == "inactive" || state == "failed" {
		ts = props.InactiveEnterTimestamp
	} else if state == "activating" {
		ts = props.InactiveExitTimestamp
	} else {
		ts = props.ActiveExitTimestamp
	}

	//That second number is "USEC_INFINITY" which seems to a thing systemd cares about
	if ts <= 0 || ts == 18446744073709551615 {
		return 0, nil
	}

	//convert from usec
	return int64(ts) * 1000, nil

}
