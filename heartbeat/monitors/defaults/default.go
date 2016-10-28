package defaults

import (
	// register standard active monitors
	_ "github.com/elastic/beats/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/heartbeat/monitors/active/tcp"
)
