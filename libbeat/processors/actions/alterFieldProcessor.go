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
	FullPath      bool

	processorName string
	alterFunc     alterFieldFunc
}

// NewAlterFieldProcessor is an umbrella method for processing events based on provided fields. Such as converting event keys to uppercase/lowercase
func NewAlterFieldProcessor(c *conf.C, processorName string, alterFunc alterFieldFunc) (processors.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
		FailOnError   bool     `config:"fail_on_error"`
		FullPath      bool     `config:"fail_path"`
	}{
		IgnoreMissing: false,
		FailOnError:   true,
		FullPath:      false,
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
		FullPath:      config.FullPath,
		processorName: processorName,
		alterFunc:     changeFunc,
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
	// Get the value of the field to alter
	value, err := event.GetValue(field)
	if err != nil {
		if a.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, err)
	}

	// Delete the exisiting value
	if strings.ContainsRune(field, '.') && a.FullPath {
		// In case of full_path set to true, we need to make sure to modify all the keys in the chain
		firstField := field[:strings.Index(field, ".")]
		if err := event.Delete(firstField); err != nil {
			return fmt.Errorf("could not delete key: %s, Error: %v", field, err)
		}
	} else {
		if err := event.Delete(field); err != nil {
			return fmt.Errorf("could not delete key: %s, Error: %v", field, err)
		}
	}

	// Alter the field
	var alterString string
	if strings.ContainsRune(field, '.') && !a.FullPath {
		// In case of nested fields provided, we need to make sure to only modify the latest key in the chain
		lastIndexRuneFunc := func(r rune) bool { return r == '.' }
		idx := strings.LastIndexFunc(field, lastIndexRuneFunc)
		alterString = field[:idx+1] + a.alterFunc(field[idx+1:])
	} else {
		alterString = a.alterFunc(field)
	}

	// Put the field back
	if _, err := event.PutValue(alterString, value); err != nil {
		return fmt.Errorf("could not put value: %s: %v, Error: %v", alterString, value, err)
	}

	return nil
}
