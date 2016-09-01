package filebeat

import (
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstriface"
)

var schema = s.Schema{
	"harvesters": s.Object{
		"started": c.Int("filebeat.harvester.started"),
		"closed":  c.Int("filebeat.harvester.closed"),
		"running": c.Int("filebeat.harvester.running"),
		"skipped": c.Int("filebeat.harvester.skipped"),
		"files": s.Object{
			"open":      c.Int("filebeat.harvester.open_files"),
			"truncated": c.Int("filebeat.harvester.files.truncated"),
		},
	},
	"prospectors": s.Object{
		"log_files": s.Object{
			"renamed":   c.Int("filebeat.prospector.log.files.renamed"),
			"truncated": c.Int("filebeat.prospector.log.files.truncated"),
		},
	},
	"registrar": s.Object{
		"states": s.Object{
			"update":  c.Int("registar.states.update"),
			"cleanup": c.Int("registar.states.cleanup"),
			"current": c.Int("registar.states.current"),
		},
		"writes": c.Int("registrar.writes"),
	},
}
