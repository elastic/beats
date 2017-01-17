package stats

import (
	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/metricbeat/schema"
	c "github.com/elastic/beats/metricbeat/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"notifications": s.Object{
			"queue_length": c.Int("prometheus_notifications_queue_length"),
			"dropped":      c.Int("prometheus_notifications_dropped_total"),
		},
		"processes": s.Object{
			"open_fds": c.Int("process_open_fds"),
		},
		"storage": s.Object{
			"chunks_to_persist": c.Int("prometheus_local_storage_chunks_to_persist"),
		},
	}
)

func eventMapping(entries map[string]interface{}) (common.MapStr, error) {
	return schema.Apply(entries), nil
}
