package beater

import (
	libevents "github.com/elastic/beats/v7/libbeat/beat/events"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
)

type evaluationResultParser struct {
}

func NewEvaluationResultParser() (*evaluationResultParser, error) {

	return &evaluationResultParser{}, nil
}

func (parser *evaluationResultParser) ParseResult(index, result interface{}, uuid uuid.UUID, timestamp time.Time) ([]beat.Event, error) {

	events := make([]beat.Event, 0)
	var opaResultMap = result.(map[string]interface{})
	var opaResult RuleResult
	err := mapstructure.Decode(opaResultMap, &opaResult)

	if err != nil {
		return nil, err
	}

	for _, finding := range opaResult.Findings {
		event := beat.Event{
			Timestamp: timestamp,
			Fields: common.MapStr{
				"run_id":   uuid,
				"result":   finding.Result,
				"resource": opaResult.Resource,
				"rule":     finding.Rule,
			},
		}
		// Insert datastream as index to event struct
	if index != "" {

		event.Meta = common.MapStr{libevents.FieldMetaIndex: index}
	}

		events = append(events, event)
	}

	return events, err
}

type RuleResult struct {
	Findings []Finding   `json:"findings"`
	Resource interface{} `json:"resource"`
}

type Finding struct {
	Result interface{} `json:"result"`
	Rule   interface{} `json:"rule"`
}
