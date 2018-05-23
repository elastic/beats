package queue

import (
	"encoding/json"

	"github.com/joeshaw/multierror"

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
				"details": c.Dict("messages_details", s.Schema{
					"rate": c.Float("rate"),
				}),
			},
			"ready": s.Object{
				"count": c.Int("messages_ready"),
				"details": c.Dict("messages_ready_details", s.Schema{
					"rate": c.Float("rate"),
				}),
			},
			"unacknowledged": s.Object{
				"count": c.Int("messages_unacknowledged"),
				"details": c.Dict("messages_unacknowledged_details", s.Schema{
					"rate": c.Float("rate"),
				}),
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
				"count": c.Int("disk_reads", s.Optional),
			},
			"writes": s.Object{
				"count": c.Int("disk_writes", s.Optional),
			},
		},
	}
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var queues []map[string]interface{}
	err := json.Unmarshal(content, &queues)
	if err != nil {
		logp.Err("Error: ", err)
		return nil, err
	}

	var events []common.MapStr
	var errors multierror.Errors

	for _, queue := range queues {
		event, err := eventMapping(queue)
		events = append(events, event)
		errors = append(errors, err)
	}

	return events, errors.Err()
}

func eventMapping(queue map[string]interface{}) (common.MapStr, error) {
	return schema.Apply(queue)
}
