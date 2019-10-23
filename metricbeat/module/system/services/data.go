package services

import (
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/elastic/beats/libbeat/common"
	"github.com/pkg/errors"
)

// getProperties gets properties for the systemd service and returns a MapStr with useful data
func getProperties(unit dbus.UnitStatus, conn *dbus.Conn) (common.MapStr, error) {
	props, err := conn.GetAllProperties(unit.Name)
	if err != nil {
		return nil, errors.Wrap(err, "error getting properties for service")
	}

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
	if props["ExecMainCode"].(int32) > 0 {
		event["exec_code"] = translateChild(props["ExecMainCode"].(int32))
		event["exec_rc"] = props["ExecMainStatus"].(int32)
	}

	//only send timestamp if it's valid
	if timeSince != 0 {
		event["state_since"] = time.Unix(0, timeSince)
	}

	//only prints status data if we have a PID that has exited
	if props["ExecMainPID"].(uint32) > 0 {
		event["main_pid"] = props["ExecMainPID"].(uint32)
	}

	return event, nil
}

// getMetricsFromServivce checks what accounting we have enabled and uses that to determine what metrics we can send back to the user
func getMetricsFromServivce(props map[string]interface{}) common.MapStr {
	metrics := common.MapStr{}
	//This error checking is because we don't _quite_ trust the maps we get back from the API.
	if cpu, ok := props["CPUAccounting"]; ok && cpu.(bool) {
		metrics["cpu"] = common.MapStr{
			"usage": common.MapStr{
				"nsec": props["CPUUsageNSec"],
			},
		}
	}

	if mem, ok := props["MemoryAccounting"]; ok && mem.(bool) {
		metrics["memory"] = common.MapStr{
			"usage": common.MapStr{
				"bytes": props["MemoryCurrent"],
			},
		}
	}

	if tasks, ok := props["TasksAccounting"]; ok && tasks.(bool) {
		metrics["tasks"] = common.MapStr{
			"count": props["TasksCurrent"],
		}
	}

	if ip, ok := props["IPAccounting"]; ok && ip.(bool) {
		metrics["network"] = common.MapStr{
			"bytes": common.MapStr{
				"in":  props["IPIngressBytes"],
				"out": props["IPEgressBytes"],
			},
			"packets": common.MapStr{
				"in":  props["IPIngressPackets"],
				"out": props["IPEgressPackets"],
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
	}
	return "unknown"
}

// timeSince emulates the behavior of `systemctl status` with regards to reporting time:
// "Active: inactive (dead) since Sat 2019-10-19 16:31:25 PDT;""
// The dbus properties data contains a number of different timestamps, which one systemd reports depends on the unit state
func timeSince(props map[string]interface{}, state string) (int64, error) {

	var ts uint64
	//normally we would want to check these values before we cast,
	//but checking the state before we access `props` insures the values will be there.
	if state == "active" || state == "reloading" {
		ts = props["ActiveEnterTimestamp"].(uint64)
	} else if state == "inactive" || state == "failed" {
		ts = props["InactiveEnterTimestamp"].(uint64)
	} else if state == "activating" {
		ts = props["InactiveExitTimestamp"].(uint64)
	} else {
		ts = props["ActiveExitTimestamp"].(uint64)
	}

	//That second number is "USEC_INFINITY" which seems to a thing systemd cares about
	if ts <= 0 || ts == 18446744073709551615 {
		return 0, nil
	}

	//convert from usec
	return int64(ts) * 1000, nil

}
