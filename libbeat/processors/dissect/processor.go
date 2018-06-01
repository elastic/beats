package dissect

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type processor struct {
	config config
}

func init() {
	processors.RegisterPlugin("dissect", newProcessor)
}

func newProcessor(c *common.Config) (processors.Processor, error) {
	config := defaultConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}
	p := &processor{config: config}

	return p, nil
}

// Run takes the event and will apply the tokenizer on the configured field.
func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	v, err := event.GetValue(p.config.Field)
	if err != nil {
		return event, err
	}

	s, ok := v.(string)
	if !ok {
		return event, fmt.Errorf("field is not a string, value: `%v`, field: `%s`", v, p.config.Field)
	}

	m, err := p.config.Tokenizer.Dissect(s)
	if err != nil {
		return event, err
	}

	event, err = p.mapper(event, mapToMapStr(m))
	if err != nil {
		return event, err
	}

	return event, nil
}

func (p *processor) mapper(event *beat.Event, m common.MapStr) (*beat.Event, error) {
	copy := event.Fields.Clone()

	prefix := ""
	if p.config.TargetPrefix != "" {
		prefix = p.config.TargetPrefix + "."
	}
	var prefixKey string
	for k, v := range m {
		prefixKey = prefix + k
		if _, err := event.GetValue(prefixKey); err == common.ErrKeyNotFound {
			event.PutValue(prefixKey, v)
		} else {
			event.Fields = copy
			// When the target key exists but is a string instead of a map.
			if err != nil {
				return event, errors.Wrapf(err, "cannot override existing key with `%s`", prefixKey)
			}
			return event, fmt.Errorf("cannot override existing key with `%s`", prefixKey)
		}
	}

	return event, nil
}

func (p *processor) String() string {
	return "dissect=" + p.config.Tokenizer.Raw() +
		",field=" + p.config.Field +
		",target_prefix=" + p.config.TargetPrefix
}

func mapToMapStr(m Map) common.MapStr {
	newMap := make(common.MapStr, len(m))
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}
