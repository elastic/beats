package rabbitmq

import (
	"github.com/elastic/beats/metricbeat/mb/parse"
)

// Subpaths to management plugin endpoints
const (
	ConnectionsPath = "/api/connections"
	ExchangesPath   = "/api/exchanges"
	NodesPath       = "/api/nodes"
	OverviewPath    = "/api/overview"
	QueuesPath      = "/api/queues"
)

const (
	defaultScheme = "http"
	pathConfigKey = "management_path_prefix"
)

var (
	// HostParser parses host urls for RabbitMQ management plugin
	HostParser = parse.URLHostParserBuilder{
		DefaultScheme:   defaultScheme,
		PathConfigKey:   pathConfigKey,
		DefaultUsername: "guest",
		DefaultPassword: "guest",
	}.Build()
)
