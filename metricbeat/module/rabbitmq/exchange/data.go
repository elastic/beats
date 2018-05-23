package exchange

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
		"type":        c.Str("type"),
		"durable":     c.Bool("durable"),
		"auto_delete": c.Bool("auto_delete"),
		"internal":    c.Bool("internal"),
		"arguments":   c.Dict("arguments", s.Schema{}),
		"user":        c.Str("user_who_performed_action", s.Optional),
		"messages": c.Dict("message_stats", s.Schema{
			"publish_in": s.Object{
				"count": c.Int("publish_in", s.Optional),
				"details": c.Dict("publish_in_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
			"publish_out": s.Object{
				"count": c.Int("publish_out", s.Optional),
				"details": c.Dict("publish_out_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
		}, c.DictOptional),
	}
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var exchanges []map[string]interface{}
	err := json.Unmarshal(content, &exchanges)
	if err != nil {
		logp.Err("Error: ", err)
		return nil, err
	}

	events := []common.MapStr{}
	errors := s.NewErrors()

	for _, exchange := range exchanges {
		event, errs := eventMapping(exchange)
		events = append(events, event)
		errors.AddErrors(errs)
	}

	return events, errors
}

func eventMapping(exchange map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(exchange)
}
