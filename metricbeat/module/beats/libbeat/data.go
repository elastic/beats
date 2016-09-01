package libbeat

import (
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var schema = s.Schema{
	"output": s.Object{
		"elasticsearch": s.Object{
			"events": s.Object{
				"ack":     c.Int("libbeat.es.published_and_acked_events"),
				"not_ack": c.Int("libbeat.es.published_but_not_acked_events"),
			},
			"read": s.Object{
				"bytes":  c.Int("libbeat.es.publish.read_bytes"),
				"errors": c.Int("libbeat.es.publish.read_errors"),
			},
			"write": s.Object{
				"bytes":  c.Int("libbeat.es.publish.write_bytes"),
				"errors": c.Int("libbeat.es.publish.write_errors"),
			},
		},
	},
	"publisher": s.Object{
		"events": s.Object{
			"published": c.Int("libbeat.publisher.published_events"),
		},
	},
}
