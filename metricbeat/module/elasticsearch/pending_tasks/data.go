package pending_tasks

import (
	"encoding/json"
	"math"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

const (
	priorityUrgent = "URGENT"
	priorityHigh   = "HIGH"
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

type task struct {
	Priority          string  `json:"priority"`
	TimeInQueueMillis float64 `json:"time_in_queue_millis"`
}

func eventMapping(content []byte) (common.MapStr, error) {
	tasksStruct := struct {
		Tasks []task `json:"tasks"`
	}{}

	if err := json.Unmarshal(content, &tasksStruct); err != nil {
		return nil, err
	}

	maxTimeInQueueMillis := 0.
	nbTasksByPriority := make(map[string]int)
	for _, task := range tasksStruct.Tasks {
		nbTasksByPriority[task.Priority]++
		maxTimeInQueueMillis = math.Max(maxTimeInQueueMillis, task.TimeInQueueMillis)
	}

	return common.MapStr{
		"pending_tasks_total":           len(tasksStruct.Tasks),
		"pending_tasks_priority_urgent": nbTasksByPriority[priorityUrgent],
		"pending_tasks_priority_high":   nbTasksByPriority[priorityHigh],
		"max_time_in_queue_millis":      maxTimeInQueueMillis,
	}, nil
}

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
