package instance

import (
	_ "github.com/elastic/beats/libbeat/autodiscover/providers/docker" // Register autodiscover providers
	_ "github.com/elastic/beats/libbeat/autodiscover/providers/jolokia"
	_ "github.com/elastic/beats/libbeat/autodiscover/providers/kubernetes"
	_ "github.com/elastic/beats/libbeat/monitoring/report/elasticsearch" // Register default monitoring reporting
	_ "github.com/elastic/beats/libbeat/processors/actions"              // Register default processors.
	_ "github.com/elastic/beats/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_docker_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_host_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
	_ "github.com/elastic/beats/libbeat/processors/add_locale"
	_ "github.com/elastic/beats/libbeat/processors/add_process_metadata"
	_ "github.com/elastic/beats/libbeat/processors/dissect"
	_ "github.com/elastic/beats/libbeat/processors/dns"
	_ "github.com/elastic/beats/libbeat/publisher/includes" // Register publisher pipeline modules
)
