package pending_tasks

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"insert_order":         c.Int("insert_order"),
		"priority":             c.Str("priority"),
		"source":               c.Str("source"),
		"time_in_queue_millis": c.Int("time_in_queue_millis"),
		"time_in_queue":        c.Str("time_in_queue"),
	}
)

func eventsMapping(content []byte) ([]common.MapStr, error) {
	tasksStruct := struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}{}

	if err := json.Unmarshal(content, &tasksStruct); err != nil {
		return nil, err
	}

	var events []common.MapStr
	errors := s.NewErrors()

	for _, task := range tasksStruct.Tasks {
		event, errs := eventMapping(task)
		errors.AddErrors(errs)
		events = append(events, event)
	}

	return events, errors
}

func eventMapping(task map[string]interface{}) (common.MapStr, *s.Errors) {
	return schema.Apply(task)
}
