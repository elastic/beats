package actions

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
		FullPath      bool     `config:"full_path"`
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
	// Get the value of the field to alter

	key, value := getcaseInsensitiveValue(event.Fields, field)
	if value == nil {
		if a.IgnoreMissing {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, mapstr.ErrKeyNotFound)
	}

	// Delete the exisiting value
	if err := event.Delete(key); err != nil {
		return fmt.Errorf("could not delete key: %s, Error: %v", key, err)
	}

	// Alter the field
	var alterString string
	if strings.ContainsRune(key, '.') {
		// In case of nested fields provided, we need to make sure to only modify the latest key in the chain
		lastIndexRuneFunc := func(r rune) bool { return r == '.' }
		idx := strings.LastIndexFunc(key, lastIndexRuneFunc)
		alterString = key[:idx+1] + a.alterFunc(key[idx+1:])
	} else {
		alterString = a.alterFunc(key)
	}

	// Put the field back
	if _, err := event.PutValue(alterString, value); err != nil {
		return fmt.Errorf("could not put value: %s: %v, Error: %v", alterString, value, err)
	}

	return nil
}

func getcaseInsensitiveValue(event mapstr.M, field string) (fi string, va interface{}) {

	// Fast path, key is present as is.
	if v, err := event.GetValue(field); err == nil {
		return field, v
	}

	// iterate through the map for case insensitive search
	subkey := strings.Split(field, ".")
	data := event

	// outer function goes through all the processor fields seperated by '.'
	for i, key := range subkey {
		keyfound := false
		for jsonKey, jsonValue := range data {
			if strings.EqualFold(jsonKey, key) {
				keyfound = true
				va = jsonValue
				subkey[i] = jsonKey
				break
			}
		}

		if keyfound {
			keyfound = false
			data, _ = toMapStr(va)
		} else {
			return field, nil
		}
	}

	return strings.Join(subkey, "."), va
}

// toMapStr performs a type assertion on v and returns a MapStr. v can be either
// a MapStr or a map[string]interface{}. If it's any other type or nil then
// an error is returned.
func toMapStr(v interface{}) (mapstr.M, error) {
	m, ok := tryToMapStr(v)
	if !ok {
		return nil, fmt.Errorf("expected map but type is %T", v)
	}
	return m, nil
}

func tryToMapStr(v interface{}) (mapstr.M, bool) {
	switch m := v.(type) {
	case mapstr.M:
		return m, true
	case map[string]interface{}:
		return mapstr.M(m), true
	default:
		return nil, false
	}
}