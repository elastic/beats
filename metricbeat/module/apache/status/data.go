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
		totalAccess            int
		totalKB                int
		uptime                 int
		cpu_load               float64
		cpu_user               float64
		cpu_system             float64
		cpu_children_user      float64
		cpu_children_system    float64
		req_per_sec            float64
		bytes_per_sec          float64
		bytes_per_req          float64
		busy_workers           int
		idle_workers           int
		conns_total            int
		conns_async_writing    int
		conns_async_keep_alive int
		conns_async_closing    int
		server_uptime_seconds  int
		load1                  float64
		load5                  float64
		load15                 float64
		tot_s                  int
		tot_r                  int
		tot_w                  int
		tot_k                  int
		tot_d                  int
		tot_c                  int
		tot_l                  int
		tot_g                  int
		tot_i                  int
		tot_dot                int
		tot_underscore         int
		tot_total              int
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
			totalAccess, _ = strconv.Atoi(matches[1])
		}

		//Total kBytes: 12988
		re = regexp.MustCompile("Total kBytes: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			totalKB, _ = strconv.Atoi(matches[1])
		}

		// Uptime: 3229728
		re = regexp.MustCompile("Uptime: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			uptime, _ = strconv.Atoi(matches[1])
		}

		// CPULoad: .000408393
		re = regexp.MustCompile("CPULoad: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpu_load = ParseMatchFloat(matches[1], hostname, "cpu_load")
		}

		// CPUUser: 0
		re = regexp.MustCompile("CPUUser: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpu_user = ParseMatchFloat(matches[1], hostname, "cpu_user")
		}

		// CPUSystem: .01
		re = regexp.MustCompile("CPUSystem: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpu_system = ParseMatchFloat(matches[1], hostname, "cpu_system")
		}

		// CPUChildrenUser: 0
		re = regexp.MustCompile("CPUChildrenUser: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpu_children_user = ParseMatchFloat(matches[1], hostname, "cpu_children_user")
		}

		// CPUChildrenSystem: 0
		re = regexp.MustCompile("CPUChildrenSystem: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			cpu_children_system = ParseMatchFloat(matches[1], hostname, "cpu_children_system")
		}

		// ReqPerSec: .00499949
		re = regexp.MustCompile("ReqPerSec: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			req_per_sec = ParseMatchFloat(matches[1], hostname, "req_per_sec")
		}

		// BytesPerSec: 4.1179
		re = regexp.MustCompile("BytesPerSec: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			bytes_per_sec = ParseMatchFloat(matches[1], hostname, "bytes_per_sec")
		}

		// BytesPerReq: 823.665
		re = regexp.MustCompile("BytesPerReq: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			bytes_per_req = ParseMatchFloat(matches[1], hostname, "bytes_per_req")
		}

		// BusyWorkers: 1
		re = regexp.MustCompile("BusyWorkers: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			busy_workers, _ = strconv.Atoi(matches[1])
		}

		// IdleWorkers: 8
		re = regexp.MustCompile("IdleWorkers: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			idle_workers, _ = strconv.Atoi(matches[1])
		}

		// ConnsTotal: 4940
		re = regexp.MustCompile("ConnsTotal: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			conns_total, _ = strconv.Atoi(matches[1])
		}

		// ConnsAsyncWriting: 527
		re = regexp.MustCompile("ConnsAsyncWriting: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			conns_async_writing, _ = strconv.Atoi(matches[1])
		}

		// ConnsAsyncKeepAlive: 1321
		re = regexp.MustCompile("ConnsAsyncKeepAlive: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			conns_async_keep_alive, _ = strconv.Atoi(matches[1])
		}

		// ConnsAsyncClosing: 2785
		re = regexp.MustCompile("ConnsAsyncClosing: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			conns_async_closing, _ = strconv.Atoi(matches[1])
		}

		// ServerUptimeSeconds: 43
		re = regexp.MustCompile("ServerUptimeSeconds: (\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			server_uptime_seconds, _ = strconv.Atoi(matches[1])
		}

		//Load1: 0.01
		re = regexp.MustCompile("Load1: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			load1 = ParseMatchFloat(matches[1], hostname, "load1")
		}

		//Load5: 0.10
		re = regexp.MustCompile("Load5: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			load5 = ParseMatchFloat(matches[1], hostname, "load5")
		}

		//Load15: 0.06
		re = regexp.MustCompile("Load15: (\\d*.*\\d+)")
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			load15 = ParseMatchFloat(matches[1], hostname, "load15")
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

			tot_underscore = strings.Count(scr[1], "_")
			tot_s = strings.Count(scr[1], "S")
			tot_r = strings.Count(scr[1], "R")
			tot_w = strings.Count(scr[1], "W")
			tot_k = strings.Count(scr[1], "K")
			tot_d = strings.Count(scr[1], "D")
			tot_c = strings.Count(scr[1], "C")
			tot_l = strings.Count(scr[1], "L")
			tot_g = strings.Count(scr[1], "G")
			tot_i = strings.Count(scr[1], "I")
			tot_dot = strings.Count(scr[1], ".")
			tot_total = tot_underscore + tot_s + tot_r + tot_w + tot_k + tot_d + tot_c + tot_l + tot_g + tot_i + tot_dot
		}
	}

	event := common.MapStr{
		"total_access":               totalAccess,
		"total_kb":                   totalKB,
		"uptime":                     uptime,
		"cpu_load":                   cpu_load,
		"cpu_user":                   cpu_user,
		"cpu_system":                 cpu_system,
		"cpu_children_user":          cpu_children_user,
		"cpu_children_system":        cpu_children_system,
		"req_per_sec":                req_per_sec,
		"bytes_per_sec":              bytes_per_sec,
		"bytes_per_req":              bytes_per_req,
		"busy_workers":               busy_workers,
		"idle_workers":               idle_workers,
		"conns_total":                conns_total,
		"conns_async_writing":        conns_async_writing,
		"conns_async_keep_alive":     conns_async_keep_alive,
		"conns_async_closing":        conns_async_closing,
		"server_uptime_seconds":      server_uptime_seconds,
		"load1":                      load1,
		"load5":                      load5,
		"load15":                     load15,
		"scb_starting_up":            tot_s,
		"scb_reading_request":        tot_r,
		"scb_sending_reply":          tot_w,
		"scb_keepalive":              tot_k,
		"scb_dns_lookup":             tot_d,
		"scb_closing_connection":     tot_c,
		"scb_logging":                tot_l,
		"scb_gracefully_finishing":   tot_g,
		"scb_idle_cleanup":           tot_i,
		"scb_open_slot":              tot_dot,
		"scb_waiting_for_connection": tot_underscore,
		"scb_total":                  tot_total,
		"hostname":                   hostname,
	}

	return event

}

func ParseMatchFloat(inputString, hostname, fieldName string) float64 {
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
