package pending_tasks

import (
	"encoding/json"

	"github.com/joeshaw/multierror"

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
	var errors multierror.Errors

	for _, task := range tasksStruct.Tasks {
		event, err := schema.Apply(task)
		if err != nil {
			errors = append(errors, err)
		}
		events = append(events, event)
	}

	return events, errors.Err()
}
