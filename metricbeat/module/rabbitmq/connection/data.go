package connection

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
		"user":        c.Str("user"),
		"node":        c.Str("node"),
		"channels":    c.Int("channels"),
		"channel_max": c.Int("channel_max"),
		"frame_max":   c.Int("frame_max"),
		"type":        c.Str("type"),
		"packet_count": s.Object{
			"sent":     c.Int("send_cnt"),
			"received": c.Int("recv_cnt"),
			"pending":  c.Int("send_pend"),
		},
		"octet_count": s.Object{
			"sent":     c.Int("send_oct"),
			"received": c.Int("recv_oct"),
		},
		"host": c.Str("host"),
		"port": c.Int("port"),
		"peer": s.Object{
			"host": c.Str("peer_host"),
			"port": c.Int("peer_port"),
		},
	}
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var connections []map[string]interface{}
	err := json.Unmarshal(content, &connections)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}
	errors := s.NewErrors()

	for _, node := range connections {
		event, errs := eventMapping(node)
		events = append(events, event)
		errors.AddErrors(errs)

	}

	return events, errors
}

func eventMapping(connection map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(connection)
}
