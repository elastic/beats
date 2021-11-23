package beater

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	"time"
)

type evaluationResultParser struct {
}

func NewEvaluationResultParser() (*evaluationResultParser, error) {

	return &evaluationResultParser{}, nil
}

func (parser *evaluationResultParser) ParseResult(result interface{}, uuid uuid.UUID, timestamp time.Time) ([]beat.Event, error) {

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
