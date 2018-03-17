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
		"arguments":   c.Dict("arguments", s.Schema{
		}),
		"messages":    c.Dict("message_stats", s.Schema{
			"publish": s.Object{
				"count": c.Int("publish", s.Optional),
				"details": c.Dict("publish_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
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
			"ack": s.Object{
				"count": c.Int("ack", s.Optional),
				"details": c.Dict("ack_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
			"deliver_get": s.Object{
				"count": c.Int("deliver_get", s.Optional),
				"details": c.Dict("deliver_get_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
			"confirm": s.Object{
				"count": c.Int("confirm", s.Optional),
				"details": c.Dict("confirm_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
			"return_unroutable": s.Object{
				"count": c.Int("return_unroutable", s.Optional),
				"details": c.Dict("return_unroutable_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
			"redeliver": s.Object{
				"count": c.Int("redeliver", s.Optional),
				"details": c.Dict("redeliver_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
		}, c.DictOptional),
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
