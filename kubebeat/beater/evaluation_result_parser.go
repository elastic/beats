package beater

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	libevents "github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
)

type evaluationResultParser struct {
	index string
}

func NewEvaluationResultParser(index string) (*evaluationResultParser, error) {
	return &evaluationResultParser{index: index}, nil
}

func (parser *evaluationResultParser) ParseResult(result interface{}, cycleId uuid.UUID) ([]beat.Event, error) {
	events := make([]beat.Event, 0)
	var opaResultMap = result.(map[string]interface{})
	var opaResult RuleResult
	err := mapstructure.Decode(opaResultMap, &opaResult)

	if err != nil {
		return nil, err
	}

	timestamp := time.Now()
	for _, finding := range opaResult.Findings {
		event := beat.Event{
			Timestamp: timestamp,
			Fields: common.MapStr{
				"cycle_id": cycleId,
				"result":   finding.Result,
				"resource": opaResult.Resource,
				"rule":     finding.Rule,
			},
		}
		// Insert datastream as index to event struct
		event.Meta = common.MapStr{libevents.FieldMetaIndex: parser.index}

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
