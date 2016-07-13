package status

import (
	"bufio"
	"io"
	"regexp"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	h "github.com/elastic/beats/metricbeat/helper"
)

var (
	scoreboardRegexp = regexp.MustCompile("(Scoreboard):\\s+((_|S|R|W|K|D|C|L|G|I|\\.)+)")

	// This should match: "CPUSystem: .01"
	matchNumber = regexp.MustCompile("(^[0-9a-zA-Z ]+):\\s+(\\d*\\.?\\d+)")
)

// Map body to MapStr
func eventMapping(body io.ReadCloser, hostname string) common.MapStr {
	var (
		totalS          int
		totalR          int
		totalW          int
		totalK          int
		totalD          int
		totalC          int
		totalL          int
		totalG          int
		totalI          int
		totalDot        int
		totalUnderscore int
		totalAll        int
	)

	fullEvent := map[string]string{}
	scanner := bufio.NewScanner(body)

	// Iterate through all events to gather data
	for scanner.Scan() {
		if match := matchNumber.FindStringSubmatch(scanner.Text()); len(match) == 3 {
			// Total Accesses: 16147
			//Total kBytes: 12988
			// Uptime: 3229728
			// CPULoad: .000408393
			// CPUUser: 0
			// CPUSystem: .01
			// CPUChildrenUser: 0
			// CPUChildrenSystem: 0
			// ReqPerSec: .00499949
			// BytesPerSec: 4.1179
			// BytesPerReq: 823.665
			// BusyWorkers: 1
			// IdleWorkers: 8
			// ConnsTotal: 4940
			// ConnsAsyncWriting: 527
			// ConnsAsyncKeepAlive: 1321
			// ConnsAsyncClosing: 2785
			// ServerUptimeSeconds: 43
			//Load1: 0.01
			//Load5: 0.10
			//Load15: 0.06
			fullEvent[match[1]] = match[2]

		} else if match := scoreboardRegexp.FindStringSubmatch(scanner.Text()); len(match) == 4 {
			// Scoreboard Key:
			// "_" Waiting for Connection, "S" Starting up, "R" Reading Request,
			// "W" Sending Reply, "K" Keepalive (read), "D" DNS Lookup,
			// "C" Closing connection, "L" Logging, "G" Gracefully finishing,
			// "I" Idle cleanup of worker, "." Open slot with no current process
			// Scoreboard: _W____........___...............................................................................................................................................................................................................................................

			totalUnderscore = strings.Count(match[2], "_")
			totalS = strings.Count(match[2], "S")
			totalR = strings.Count(match[2], "R")
			totalW = strings.Count(match[2], "W")
			totalK = strings.Count(match[2], "K")
			totalD = strings.Count(match[2], "D")
			totalC = strings.Count(match[2], "C")
			totalL = strings.Count(match[2], "L")
			totalG = strings.Count(match[2], "G")
			totalI = strings.Count(match[2], "I")
			totalDot = strings.Count(match[2], ".")
			totalAll = totalUnderscore + totalS + totalR + totalW + totalK + totalD + totalC + totalL + totalG + totalI + totalDot
		} else {
			debugf("Unexpected line in apache server-status output: %s", scanner.Text())
		}
	}

	errs := map[string]error{}
	event := common.MapStr{
		"hostname":          hostname,
		"total_accesses":    h.ToInt("Total Accesses", fullEvent, errs, "total_accesses"),
		"total_kbytes":      h.ToInt("Total kBytes", fullEvent, errs, "total_kbytes"),
		"requests_per_sec":  h.ToFloat("ReqPerSec", fullEvent, errs, "requests_per_sec"),
		"bytes_per_sec":     h.ToFloat("BytesPerSec", fullEvent, errs, "bytes_per_sec"),
		"bytes_per_request": h.ToFloat("BytesPerReq", fullEvent, errs, "bytes_per_request"),
		"workers": common.MapStr{
			"busy": h.ToInt("BusyWorkers", fullEvent, errs, "workers.busy"),
			"idle": h.ToInt("IdleWorkers", fullEvent, errs, "workers.idle"),
		},
		"uptime": common.MapStr{
			"server_uptime": h.ToInt("ServerUptimeSeconds", fullEvent, errs, "uptime.server_uptime"),
			"uptime":        h.ToInt("Uptime", fullEvent, errs, "uptime.uptime"),
		},
		"cpu": common.MapStr{
			"load":            h.ToFloat("CPULoad", fullEvent, errs, "cpu.load"),
			"user":            h.ToFloat("CPUUser", fullEvent, errs, "cpu.user"),
			"system":          h.ToFloat("CPUSystem", fullEvent, errs, "cpu.system"),
			"children_user":   h.ToFloat("CPUChildrenUser", fullEvent, errs, "cpu.children_user"),
			"children_system": h.ToFloat("CPUChildrenSystem", fullEvent, errs, "cpu.children_system"),
		},
		"connections": common.MapStr{
			"total": h.ToInt("ConnsTotal", fullEvent, errs, "connections.total"),
			"async": common.MapStr{
				"writing":    h.ToInt("ConnsAsyncWriting", fullEvent, errs, "connections.async.writing"),
				"keep_alive": h.ToInt("ConnsAsyncKeepAlive", fullEvent, errs, "connections.async.keep_alive"),
				"closing":    h.ToInt("ConnsAsyncClosing", fullEvent, errs, "connections.async.closing"),
			},
		},
		"load": common.MapStr{
			"1":  h.ToFloat("Load1", fullEvent, errs, "load.1"),
			"5":  h.ToFloat("Load5", fullEvent, errs, "load.5"),
			"15": h.ToFloat("Load15", fullEvent, errs, "load.15"),
		},
		"scoreboard": common.MapStr{
			"starting_up":            totalS,
			"reading_request":        totalR,
			"sending_reply":          totalW,
			"keepalive":              totalK,
			"dns_lookup":             totalD,
			"closing_connection":     totalC,
			"logging":                totalL,
			"gracefully_finishing":   totalG,
			"idle_cleanup":           totalI,
			"open_slot":              totalDot,
			"waiting_for_connection": totalUnderscore,
			"total":                  totalAll,
		},
	}
	h.RemoveErroredKeys(event, errs)

	return event
}

/*
func parseMatchFloat(input interface{}, fieldName string) float64 {
	var parseString string

	if input != nil {
		if strings.HasPrefix(input.(string), ".") {
			parseString = strings.Replace(input.(string), ".", "0.", 1)
		} else {
			parseString = input.(string)
		}

		outputFloat, err := strconv.ParseFloat(parseString, 64)
		if err != nil {
			logp.Err("Cannot parse string '%s' to float for field '%s'. Error: %+v", input.(string), fieldName, err)
			return 0.0
		}
		return outputFloat
	} else {
		return 0.0
	}
}*/
