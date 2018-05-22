package pending_tasks

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	schema = s.Schema{
		"insert_order":     c.Int("insert_order"),
		"priority":         c.Str("priority"),
		"source":           c.Str("source"),
		"time_in_queue.ms": c.Int("time_in_queue_millis"),
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
		event, errs := schema.Apply(task)
		errors.AddErrors(errs)
		events = append(events, event)
	}

	return events, errors
}
