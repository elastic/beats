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

// Map body to MapStr
func eventMapping(body io.ReadCloser, hostname string, metricset string) common.MapStr {

	var (
		totalAccesses       int
		totalKBytes         int
		uptime              int
		cpuLoad             float64
		cpuUser             float64
		cpuSystem           float64
		cpuChildrenUser     float64
		cpuChildrenSystem   float64
		reqPerSec           float64
		bytesPerSec         float64
		bytesPerReq         float64
		busyWorkers         int
		idleWorkers         int
		connsTotal          int
		connsAsyncWriting   int
		connsAsyncKeepAlive int
		connsAsyncClosing   int
		serverUptimeSeconds int
		load1               float64
		load5               float64
		load15              float64
		totalS              int
		totalR              int
		totalW              int
		totalK              int
		totalD              int
		totalC              int
		totalL              int
		totalG              int
		totalI              int
		totalDot            int
		totalUnderscore     int
		totalAll            int
	)

	var re *regexp.Regexp

	// Reads file line by line
	scanner := bufio.NewScanner(body)

	// See https://github.com/radoondas/apachebeat/blob/master/collector/status.go#L114
	// Only as POC
	for scanner.Scan() {

		// Total Accesses: 16147
		re = regexp.MustCompile("Total Accesses: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			totalAccesses, _ = strconv.Atoi(matches[1])
		}

		//Total kBytes: 12988
		re = regexp.MustCompile("Total kBytes: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			totalKBytes, _ = strconv.Atoi(matches[1])
		}

		// Uptime: 3229728
		re = regexp.MustCompile("Uptime: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			uptime, _ = strconv.Atoi(matches[1])
		}

		// CPULoad: .000408393
		re = regexp.MustCompile("CPULoad: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpuLoad = parseMatchFloat(matches[1], hostname, "cpuLoad")
		}

		// CPUUser: 0
		re = regexp.MustCompile("CPUUser: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpuUser = parseMatchFloat(matches[1], hostname, "cpuUser")
		}

		// CPUSystem: .01
		re = regexp.MustCompile("CPUSystem: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpuSystem = parseMatchFloat(matches[1], hostname, "cpuSystem")
		}

		// CPUChildrenUser: 0
		re = regexp.MustCompile("CPUChildrenUser: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpuChildrenUser = parseMatchFloat(matches[1], hostname, "cpuChildrenUser")
		}

		// CPUChildrenSystem: 0
		re = regexp.MustCompile("CPUChildrenSystem: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpuChildrenSystem = parseMatchFloat(matches[1], hostname, "cpuChildrenSystem")
		}

		// ReqPerSec: .00499949
		re = regexp.MustCompile("ReqPerSec: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			reqPerSec = parseMatchFloat(matches[1], hostname, "reqPerSec")
		}

		// BytesPerSec: 4.1179
		re = regexp.MustCompile("BytesPerSec: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			bytesPerSec = parseMatchFloat(matches[1], hostname, "bytesPerSec")
		}

		// BytesPerReq: 823.665
		re = regexp.MustCompile("BytesPerReq: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			bytesPerReq = parseMatchFloat(matches[1], hostname, "bytesPerReq")
		}

		// BusyWorkers: 1
		re = regexp.MustCompile("BusyWorkers: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			busyWorkers, _ = strconv.Atoi(matches[1])
		}

		// IdleWorkers: 8
		re = regexp.MustCompile("IdleWorkers: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			idleWorkers, _ = strconv.Atoi(matches[1])
		}

		// ConnsTotal: 4940
		re = regexp.MustCompile("ConnsTotal: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			connsTotal, _ = strconv.Atoi(matches[1])
		}

		// ConnsAsyncWriting: 527
		re = regexp.MustCompile("ConnsAsyncWriting: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			connsAsyncWriting, _ = strconv.Atoi(matches[1])
		}

		// ConnsAsyncKeepAlive: 1321
		re = regexp.MustCompile("ConnsAsyncKeepAlive: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			connsAsyncKeepAlive, _ = strconv.Atoi(matches[1])
		}

		// ConnsAsyncClosing: 2785
		re = regexp.MustCompile("ConnsAsyncClosing: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			connsAsyncClosing, _ = strconv.Atoi(matches[1])
		}

		// ServerUptimeSeconds: 43
		re = regexp.MustCompile("ServerUptimeSeconds: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			serverUptimeSeconds, _ = strconv.Atoi(matches[1])
		}

		//Load1: 0.01
		re = regexp.MustCompile("Load1: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			load1 = parseMatchFloat(matches[1], hostname, "load1")
		}

		//Load5: 0.10
		re = regexp.MustCompile("Load5: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			load5 = parseMatchFloat(matches[1], hostname, "load5")
		}

		//Load15: 0.06
		re = regexp.MustCompile("Load15: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			load15 = parseMatchFloat(matches[1], hostname, "load15")
		}

		// Scoreboard Key:
		// "_" Waiting for Connection, "S" Starting up, "R" Reading Request,
		// "W" Sending Reply, "K" Keepalive (read), "D" DNS Lookup,
		// "C" Closing connection, "L" Logging, "G" Gracefully finishing,
		// "I" Idle cleanup of worker, "." Open slot with no current process
		// Scoreboard: _W____........___...............................................................................................................................................................................................................................................
		re = regexp.MustCompile("Scoreboard: (_|S|R|W|K|D|C|L|G|I|\\.)+")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			scr := strings.Split(scanner.Text(), " ")

			totalUnderscore = strings.Count(scr[1], "_")
			totalS = strings.Count(scr[1], "S")
			totalR = strings.Count(scr[1], "R")
			totalW = strings.Count(scr[1], "W")
			totalK = strings.Count(scr[1], "K")
			totalD = strings.Count(scr[1], "D")
			totalC = strings.Count(scr[1], "C")
			totalL = strings.Count(scr[1], "L")
			totalG = strings.Count(scr[1], "G")
			totalI = strings.Count(scr[1], "I")
			totalDot = strings.Count(scr[1], ".")
			totalAll = totalUnderscore + totalS + totalR + totalW + totalK + totalD + totalC + totalL + totalG + totalI + totalDot
		}
	}

	event := common.MapStr{
		"hostname":          hostname,
		"total_accesses":    totalAccesses,
		"total_kbytes":      totalKBytes,
		"requests_per_sec":  reqPerSec,
		"bytes_per_sec":     bytesPerSec,
		"bytes_per_request": bytesPerReq,
		"workers": common.MapStr{
			"busy": busyWorkers,
			"idle": idleWorkers,
		},
		"uptime": common.MapStr{
			"server_uptime": serverUptimeSeconds,
			"uptime":        uptime,
		},
		"cpu": common.MapStr{
			"load":            cpuLoad,
			"user":            cpuUser,
			"system":          cpuSystem,
			"children_user":   cpuChildrenUser,
			"children_system": cpuChildrenSystem,
		},
		"connections": common.MapStr{
			"total": connsTotal,
			"async": common.MapStr{
				"writing":    connsAsyncWriting,
				"keep_alive": connsAsyncKeepAlive,
				"closing":    connsAsyncClosing,
			},
		},
		"load": common.MapStr{
			"1":  load1,
			"5":  load5,
			"15": load15,
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

func parseMatchFloat(inputString, hostname, fieldName string) float64 {
	var parseString string
	if strings.HasPrefix(inputString, ".") {
		parseString = strings.Replace(inputString, ".", "0.", 1)
	} else {
		parseString = inputString
	}
	outputFloat, er := strconv.ParseFloat(parseString, 64)

	/* Do we need to log failure? */
	if er != nil {
		logp.Warn("Host: %s - cannot parse string %s: %s to float.", hostname, fieldName, inputString)
		return 0.0
	}
	return outputFloat
}
