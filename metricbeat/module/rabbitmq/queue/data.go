package queue

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	schema = s.Schema{
		"name":        c.Str("name"),
		"vhost":       c.Str("vhost"),
		"durable":     c.Bool("durable"),
		"auto_delete": c.Bool("auto_delete"),
		"exclusive":   c.Bool("exclusive"),
		"node":        c.Str("node"),
		"state":       c.Str("state"),
		"arguments": c.Dict("arguments", s.Schema{
			"max_priority": c.Int("x-max-priority", s.Optional),
		}),
		"consumers": s.Object{
			"count": c.Int("consumers"),
			"utilisation": s.Object{
				"pct": c.Int("consumer_utilisation", s.Optional),
			},
		},
		"messages": s.Object{
			"total": s.Object{
				"count": c.Int("messages"),
			},
			"ready": s.Object{
				"count": c.Int("messages_ready"),
			},
			"unacknowledged": s.Object{
				"count": c.Int("messages_unacknowledged"),
			},
			"persistent": s.Object{
				"count": c.Int("messages_persistent"),
			},
		},
		"memory": s.Object{
			"bytes": c.Int("memory"),
		},
		"disk": s.Object{
			"reads": s.Object{
				"count": c.Int("disk_reads"),
			},
			"writes": s.Object{
				"count": c.Int("disk_writes"),
			},
		},
	}
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var queues []map[string]interface{}
	err := json.Unmarshal(content, &queues)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}
	errors := s.NewErrors()

	for _, queue := range queues {
		event, errs := eventMapping(queue)
		events = append(events, event)
		errors.AddErrors(errs)
	}

	return events, errors
}

func eventMapping(queue map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(queue)
}
