package opa

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/gofrs/uuid"
)

type EventParser struct {
}

func NewEventParser() (*EventParser, error) {
	return &EventParser{}, nil
}

func (parser *EventParser) ParseResult(result interface{}, uuid uuid.UUID, timestamp time.Time) ([]beat.Event, error) {

	events := make([]beat.Event, 0)
	var opaResult = result.(map[string]interface{})

	if findings, ok := opaResult["findings"].([]interface{}); ok {
		for _, findingRaw := range findings {
			if finding, ok := findingRaw.(map[string]interface{}); ok {
				event := beat.Event{
					Timestamp: timestamp,
					Fields: common.MapStr{
						"run_id":   uuid,
						"result":   finding["result"],
						"resource": opaResult["resource"],
						"rule":     finding["rule"],
					},
				}
				events = append(events, event)
			}
		}
	}

	return events, nil
}
