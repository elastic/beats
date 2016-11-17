package stubstatus

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
)

var (
	activeRe  = regexp.MustCompile("Active connections: (\\d+)")
	requestRe = regexp.MustCompile("\\s(\\d+)\\s+(\\d+)\\s+(\\d+)")
	connRe    = regexp.MustCompile("Reading: (\\d+) Writing: (\\d+) Waiting: (\\d+)")
)

// Map body to MapStr
func eventMapping(m *MetricSet, body io.ReadCloser, hostname string, metricset string) (common.MapStr, error) {
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

	scanner := bufio.NewScanner(body)

	// Parse active connections.
	scanner.Scan()
	matches := activeRe.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return nil, fmt.Errorf("cannot parse active connections from Nginx stub status")
	}

	active, _ = strconv.Atoi(matches[1])

	// Skip request status headers.
	scanner.Scan()

	// Parse request status.
	scanner.Scan()
	matches = requestRe.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return nil, fmt.Errorf("cannot parse request status from Nginx stub status")
	}

	accepts, _ = strconv.Atoi(matches[1])
	handled, _ = strconv.Atoi(matches[2])
	requests, _ = strconv.Atoi(matches[3])

	// Derived request status.
	dropped = accepts - handled
	current = requests - m.requests

	// Kept for next run.
	m.requests = requests

	// Parse connection status.
	scanner.Scan()
	matches = connRe.FindStringSubmatch(scanner.Text())
	if matches == nil {
		return nil, fmt.Errorf("cannot parse connection status from Nginx stub status")
	}

	reading, _ = strconv.Atoi(matches[1])
	writing, _ = strconv.Atoi(matches[2])
	waiting, _ = strconv.Atoi(matches[3])

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

	return event, nil
}
