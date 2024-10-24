package actions

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/pkg/errors"
)

// fieldProccessorFunc defines how fields must be processed
type fieldProccessorFunc func(event *beat.Event, field string) error

type changeFieldProcessor struct {
	Fields        []string
	IgnoreMissing bool
	FailOnError   bool
	fieldProcess  string
	changeFunc    fieldProccessorFunc
}

// NewChangeFieldProcessor is an umbrella method for processing events based on provided fields. Such as converting event keys to uppercase/lowercase
func NewChangeFieldProcessor(c *conf.C, fieldProcess string, changeFunc fieldProccessorFunc) (processors.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
		FailOnError   bool     `config:"fail_on_error"`
	}{
		IgnoreMissing: false,
		FailOnError:   true,
	}

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %s fields configuration: %s", fieldProcess, err)
	}

	// Skip mandatory fields
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if field == readOnly {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}

	{
		return &changeFieldProcessor{
			Fields:        config.Fields,
			IgnoreMissing: config.IgnoreMissing,
			FailOnError:   config.FailOnError,
			fieldProcess:  fieldProcess,
			changeFunc:    changeFunc,
		}, nil
	}

}

func (c *changeFieldProcessor) String() string {
	return fmt.Sprintf("%s fields=%+v", c.fieldProcess, *c)
}

func (c *changeFieldProcessor) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	if c.FailOnError {
		backup = event.Clone()
	}

	for _, field := range c.Fields {
		err := c.changeFunc(event, field)

		if c.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			err = nil
		} else if errors.Is(err, mapstr.ErrKeyNotFound) {
			err = fmt.Errorf("could not fetch value for key: %s, Error: %v", field, err)
		}

		if err != nil {
			if c.FailOnError {
				event = backup
				event.PutValue("error.message", err.Error())
				return event, err
			}
		}
	}

	return event, nil
}
