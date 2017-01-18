package php_fpm

import (
	"fmt"
	"io"
	"net/http"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/status"
)

// HostParser is used for parsing the configured php-fpm hosts.
var HostParser = parse.URLHostParserBuilder{
	DefaultScheme: defaultScheme,
	DefaultPath:   defaultPath,
}.Build()

// StatsClient provides access to php-fpm stats api
type StatsClient struct {
	address  string
	user     string
	password string
	http     *http.Client
}

// NewStatsClient creates a new StatsClient
func NewStatsClient(m mb.BaseMetricSet, isFullStats bool) *StatsClient {
	var address string
	address = m.HostData().SanitizedURI + "?json"
	if isFullStats {
		address += "&full"
	}
	return &StatsClient{
		address:  address,
		user:     m.HostData().User,
		password: m.HostData().Password,
		http:     &http.Client{Timeout: m.Module().Config().Timeout},
	}
}

// Fetch php-fpm stats
func (c *StatsClient) Fetch() (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.address, nil)
	if c.user != "" || c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making http request: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

// PoolStats defines all stats fields from a php-fpm pool
type PoolStats struct {
	Pool               string `json:"pool"`
	ProcessManager     string `json:"process manager"`
	StartTime          int    `json:"start time"`
	StartSince         int    `json:"start since"`
	AcceptedConn       int    `json:"accepted conn"`
	ListenQueue        int    `json:"listen queue"`
	MaxListQueue       int    `json:"max list queue"`
	ListenQueueLen     int    `json:"listen queue len"`
	IdleProcesses      int    `json:"idle processes"`
	ActiveProcesses    int    `json:"active processes"`
	TotalProcesses     int    `json:"total processes"`
	MaxActiveProcesses int    `json:"max active processes"`
	MaxChildrenReached int    `json:"max children reached"`
	SlowRequests       int    `json:"slow requests"`
}

// ProcStats defines all stats fields from a process in php-fpm pool
type ProcStats struct {
	Pid               int     `json:"pid"`
	State             string  `json:"state"`
	StartTime         int     `json:"start time"`
	StartSince        int     `json:"start since"`
	Requests          int     `json:"requests"`
	RequestDuration   int     `json:"request duration"`
	RequestMethod     string  `json:"request method"`
	RequestURI        string  `json:"request uri"`
	ContentLength     int     `json:"content length"`
	User              string  `json:"user"`
	Script            string  `json:"script"`
	LastRequestCPU    float64 `json:"last request cpu"`
	LastRequestMemory int     `json:"last request memory"`
}

// FullStats defines all stats fields of the full stats api call (pool + processes)
type FullStats struct {
	PoolStats
	Processes []ProcStats `json:"processes"`
}
