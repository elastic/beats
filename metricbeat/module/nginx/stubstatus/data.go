package stubstatus

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

var (
	activeRe  = regexp.MustCompile("Active connections: (\\d+)")
	requestRe = regexp.MustCompile("\\s(\\d+)\\s+(\\d+)\\s+(\\d+)")
	connRe    = regexp.MustCompile("Reading: (\\d+) Writing: (\\d+) Waiting: (\\d+)")

	schema = s.Schema{
		"active":   c.Int("active"),
		"accepts":  c.Int("accepts"),
		"handled":  c.Int("handled"),
		"requests": c.Int("requests"),
		"reading":  c.Int("reading"),
		"writing":  c.Int("writing"),
		"waiting":  c.Int("waiting"),
	}
)

// Map body to MapStr
func eventMapping(m *MetricSet, body io.ReadCloser, hostname string, metricset string) (common.MapStr, error) {
	// Nginx stub status sample:
	// Active connections: 1
	// server accepts handled requests
	//  7 7 19
	// Reading: 0 Writing: 1 Waiting: 0
	var (
		active   string
		accepts  string
		handled  string
		requests string
		reading  string
		writing  string
		waiting  string
	)

	scanner := bufio.NewScanner(body)

	// Parse active connections.
	scanner.Scan()
	if matches := activeRe.FindStringSubmatch(scanner.Text()); matches == nil {
		return nil, fmt.Errorf("cannot parse active connections from Nginx stub status")
	} else {
		active = matches[1]
	}

	// Skip request status headers.
	scanner.Scan()

	// Parse request status.
	scanner.Scan()
	if matches := requestRe.FindStringSubmatch(scanner.Text()); matches == nil {
		return nil, fmt.Errorf("cannot parse request status from Nginx stub status")
	} else {
		accepts = matches[1]
		handled = matches[2]
		requests = matches[3]
	}

	// Parse connection status.
	scanner.Scan()
	if matches := connRe.FindStringSubmatch(scanner.Text()); matches == nil {
		return nil, fmt.Errorf("cannot parse connection status from Nginx stub status")
	} else {
		reading = matches[1]
		writing = matches[2]
		waiting = matches[3]
	}

	event := common.MapStr{
		"hostname": hostname,
	}
	metrics := map[string]interface{}{
		"active":   active,
		"accepts":  accepts,
		"handled":  handled,
		"requests": requests,
		"reading":  reading,
		"writing":  writing,
		"waiting":  waiting,
	}

	return schema.ApplyTo(event, metrics), nil
}
