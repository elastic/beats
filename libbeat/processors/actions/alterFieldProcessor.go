package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/pkg/errors"
)

// alterFieldFunc defines how fields must be processed
type alterFieldFunc func(field string) string

type alterFieldProcessor struct {
	Fields        []string
	IgnoreMissing bool
	FailOnError   bool

	processorName string
	alterFunc     alterFieldFunc
}

// NewAlterFieldProcessor is an umbrella method for processing events based on provided fields. Such as converting event keys to uppercase/lowercase
func NewAlterFieldProcessor(c *conf.C, processorName string, alterFunc alterFieldFunc) (processors.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
		FailOnError   bool     `config:"fail_on_error"`
	}{
		IgnoreMissing: false,
		FailOnError:   true,
	}

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %s fields configuration: %s", processorName, err)
	}

	// Skip mandatory fields
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if field == readOnly {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}

	return &alterFieldProcessor{
		Fields:        config.Fields,
		IgnoreMissing: config.IgnoreMissing,
		FailOnError:   config.FailOnError,
		processorName: processorName,
		alterFunc:     alterFunc,
	}, nil

}

func (a *alterFieldProcessor) String() string {
	return fmt.Sprintf("%s fields=%+v", a.processorName, *a)
}

func (a *alterFieldProcessor) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	if a.FailOnError {
		backup = event.Clone()
	}

	for _, field := range a.Fields {
		err := a.alter(event, field)
		if err != nil {
			if a.FailOnError {
				event = backup
				event.PutValue("error.message", err.Error())
				return event, err
			}
		}
	}

	return event, nil
}

func (a *alterFieldProcessor) alter(event *beat.Event, field string) error {

	// Get all matching keys
	allMatchingKeys := getCaseInsensitiveKeys(event.Fields, field)
	if allMatchingKeys == nil {
		if a.IgnoreMissing {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, mapstr.ErrKeyNotFound)
	}

	for _, key := range allMatchingKeys {
		// Get the value the matching key
		value, err := event.GetValue(key)
		if err != nil {
			if a.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
				return nil
			}
			return fmt.Errorf("could not fetch value for key: %s, Error: %v", key, err)
		}

		// Delete the exisiting value
		if err := event.Delete(key); err != nil {
			return fmt.Errorf("could not delete field: %s, Error: %v", key, err)
		}

		// Alter the field
		var alterString string
		if strings.ContainsRune(key, '.') {
			// In case of nested fields provided, we need to make sure to only modify the latest field in the chain
			lastIndexRuneFunc := func(r rune) bool { return r == '.' }
			idx := strings.LastIndexFunc(key, lastIndexRuneFunc)
			alterString = key[:idx+1] + a.alterFunc(key[idx+1:])

		} else {
			alterString = a.alterFunc(key)
		}

		// Put the value back
		if _, err := event.PutValue(alterString, value); err != nil {
			return fmt.Errorf("could not put value: %s: %v, Error: %v", alterString, value, err)
		}
	}

	return nil
}

func getCaseInsensitiveKeys(event mapstr.M, field string) (key []string) {
	keys := event.FlattenKeys()
	for _, k := range *keys {
		if strings.EqualFold(k, field) {
			key = append(key, k)
		}
	}
	return key
}
