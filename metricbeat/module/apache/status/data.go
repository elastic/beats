package status

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	scoreboardRegexp = regexp.MustCompile("(Scoreboard):\\s+((_|S|R|W|K|D|C|L|G|I|\\.)+)")

	// This should match: "CPUSystem: .01"
	matchNumber = regexp.MustCompile("(^[0-9a-zA-Z ]+):\\s+(\\d*\\.?\\d+)")
)

// Map body to MapStr
func eventMapping(body io.ReadCloser, hostname string, metricset string) common.MapStr {
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

	fullEvent := common.MapStr{}
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

	event := common.MapStr{
		"hostname":          hostname,
		"total_accesses":    toInt(fullEvent["Total Accesses"]),
		"total_kbytes":      toInt(fullEvent["Total kBytes"]),
		"requests_per_sec":  parseMatchFloat(fullEvent["ReqPerSec"], hostname, "ReqPerSec"),
		"bytes_per_sec":     parseMatchFloat(fullEvent["BytesPerSec"], hostname, "BytesPerSec"),
		"bytes_per_request": parseMatchFloat(fullEvent["BytesPerReq"], hostname, "BytesPerReq"),
		"workers": common.MapStr{
			"busy": toInt(fullEvent["BusyWorkers"]),
			"idle": toInt(fullEvent["IdleWorkers"]),
		},
		"uptime": common.MapStr{
			"server_uptime": toInt(fullEvent["ServerUptimeSeconds"]),
			"uptime":        toInt(fullEvent["Uptime"]),
		},
		"cpu": common.MapStr{
			"load":            parseMatchFloat(fullEvent["CPULoad"], hostname, "CPULoad"),
			"user":            parseMatchFloat(fullEvent["CPUUser"], hostname, "CPUUser"),
			"system":          parseMatchFloat(fullEvent["CPUSystem"], hostname, "CPUSystem"),
			"children_user":   parseMatchFloat(fullEvent["CPUChildrenUser"], hostname, "CPUChildrenUser"),
			"children_system": parseMatchFloat(fullEvent["CPUChildrenSystem"], hostname, "CPUChildrenSystem"),
		},
		"connections": common.MapStr{
			"total": toInt(fullEvent["ConnsTotal"]),
			"async": common.MapStr{
				"writing":    toInt(fullEvent["ConnsAsyncWriting"]),
				"keep_alive": toInt(fullEvent["ConnsAsyncKeepAlive"]),
				"closing":    toInt(fullEvent["ConnsAsyncClosing"]),
			},
		},
		"load": common.MapStr{
			"1":  parseMatchFloat(fullEvent["Load1"], hostname, "Load1"),
			"5":  parseMatchFloat(fullEvent["Load5"], hostname, "Load5"),
			"15": parseMatchFloat(fullEvent["Load15"], hostname, "Load15"),
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

	return event
}

func parseMatchFloat(input interface{}, hostname, fieldName string) float64 {
	var parseString string

	if input != nil {
		if strings.HasPrefix(input.(string), ".") {
			parseString = strings.Replace(input.(string), ".", "0.", 1)
		} else {
			parseString = input.(string)
		}
		outputFloat, er := strconv.ParseFloat(parseString, 64)

		/* Do we need to log failure? */
		if er != nil {
			debugf("Host: %s - cannot parse string %s: %s to float.", hostname, fieldName, input.(string))
			return 0.0
		}
		return outputFloat
	} else {
		return 0.0
	}
}

// toInt converts value to int. In case of error, returns 0
func toInt(param interface{}) int {
	if param == nil {
		return 0
	}

	value, err := strconv.Atoi(param.(string))

	if err != nil {
		logp.Err("Error converting param to int: %s", param)
		value = 0
	}

	return value
}
