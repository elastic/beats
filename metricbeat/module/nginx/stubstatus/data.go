package stubstatus

import (
	"bufio"
	"io"
	"regexp"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Map body to MapStr
func eventMapping(m *MetricSeter, body io.ReadCloser, hostname string, metricset string) common.MapStr {
	// Nginx stub status sample:
	// Active connections: 1
	// server accepts handled requests
	//  7 7 19
	// Reading: 0 Writing: 1 Waiting: 0
	var (
		active   int
		accepts  int
		handled  int
		dropped  int
		requests int
		current  int
		reading  int
		writing  int
		waiting  int
	)

	var re *regexp.Regexp
	scanner := bufio.NewScanner(body)

	// Parse active connections.
	scanner.Scan()
	re = regexp.MustCompile("Active connections: (\\d+)")
	if matches := re.FindStringSubmatch(scanner.Text()); matches == nil {
		logp.Warn("Fail to parse active connections from Nginx stub status")
		active = -1
	} else {
		active, _ = strconv.Atoi(matches[1])
	}

	// Skip request status headers.
	scanner.Scan()

	// Parse request status.
	scanner.Scan()
	re = regexp.MustCompile("\\s(\\d+)\\s+(\\d+)\\s+(\\d+)")
	if matches := re.FindStringSubmatch(scanner.Text()); matches == nil {
		logp.Warn("Fail to parse request status from Nginx stub status")
		accepts = -1
		handled = -1
		dropped = -1
		requests = -1
		current = -1
	} else {
		accepts, _ = strconv.Atoi(matches[1])
		handled, _ = strconv.Atoi(matches[2])
		requests, _ = strconv.Atoi(matches[3])

		// Derived request status.
		dropped = accepts - handled
		current = requests - m.requests

		// Kept for next run.
		m.requests = requests
	}

	// Parse connection status.
	scanner.Scan()
	re = regexp.MustCompile("Reading: (\\d+) Writing: (\\d+) Waiting: (\\d+)")
	var ()
	if matches := re.FindStringSubmatch(scanner.Text()); matches == nil {
		logp.Warn("Fail to parse connection status from Nginx stub status")
		reading = -1
		writing = -1
		waiting = -1
	} else {
		reading, _ = strconv.Atoi(matches[1])
		writing, _ = strconv.Atoi(matches[2])
		waiting, _ = strconv.Atoi(matches[3])
	}

	event := common.MapStr{
		"hostname": hostname,

		"active":   active,
		"accepts":  accepts,
		"handled":  handled,
		"dropped":  dropped,
		"requests": requests,
		"current":  current,
		"reading":  reading,
		"writing":  writing,
		"waiting":  waiting,
	}

	return event
}
