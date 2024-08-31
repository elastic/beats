package panos

import (
	"github.com/PaloAltoNetworks/pango"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	ModuleName = "panos"
)

type Config struct {
	HostIp    string `config:"host_ip"`
	ApiKey    string `config:"apiKey"`
	DebugMode string `config:"apiDebugMode"`
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	Config Config
	Logger *logp.Logger
	Client *pango.Firewall
}
