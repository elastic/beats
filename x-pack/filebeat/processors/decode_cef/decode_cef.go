// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/x-pack/filebeat/processors/decode_cef/cef"
)

const (
	procName = "decode_cef"
	logName  = "processor." + procName
)

func init() {
	processors.RegisterPlugin(procName, New)
}

type processor struct {
	config
	log *logp.Logger
}

// New constructs a new processor built from ucfg config.
func New(cfg *common.Config) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the "+procName+" processor configuration")
	}

	return newDecodeCEF(c)
}

func newDecodeCEF(c config) (*processor, error) {
	log := logp.NewLogger(logName)
	if c.Tag != "" {
		log = log.With("instance_id", c.Tag)
	}

	return &processor{config: c, log: log}, nil
}

func (p *processor) String() string {
	json, _ := json.Marshal(p.config)
	return procName + "=" + string(json)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	v, err := event.GetValue(p.config.Field)
	if err != nil {
		return event, nil
	}

	cefData, ok := v.(string)
	if !ok {
		return event, nil
	}

	// Ignore any leading data before the CEF header.
	idx := strings.Index(cefData, "CEF:")
	if idx == -1 {
		return event, errors.Errorf("%v field is not a CEF message: header start not found", p.config.Field)
	}
	cefData = cefData[idx:]

	// If the version < 0 after parsing then none of the data is valid so return here.
	var ce cef.Event
	if err = ce.Unpack([]byte(cefData), cef.WithFullExtensionNames()); ce.Version < 0 && err != nil {
		return event, err
	}

	cefErrors := multierr.Errors(err)
	cefObject := toCEFObject(&ce)
	event.PutValue(p.Target, cefObject)

	// Map CEF extension fields to ECS fields.
	if p.ECS {
		for key, v := range ce.Extensions {
			mapping, found := ecsKeyMapping[key]
			if !found {
				continue
			}

			// Apply translation function or use a standard type translation (e.g. string to long).
			if mapping.Translate != nil {
				translatedValue, err := mapping.Translate(v)
				if err != nil {
					cefErrors = append(cefErrors, err)
					continue
				}
				event.PutValue(mapping.Target, translatedValue)
			} else if mapping.Type != unset {
				translatedValue, err := toType(v, mapping.Type)
				if err != nil {
					cefErrors = append(cefErrors, err)
					continue
				}
				event.PutValue(mapping.Target, translatedValue)
			}
		}
	}

	// Add all parsing/conversion errors to error.message.
	for _, cefError := range cefErrors {
		if err := appendErrorMessage(event.Fields, cefError.Error()); err != nil {
			p.log.Warn("Failed adding CEF errors to event.", "error", err)
			break
		}
	}

	return event, nil
}

func toCEFObject(cefEvent *cef.Event) common.MapStr {
	// Add CEF header fields.
	cefObject := common.MapStr{"version": cefEvent.Version}
	if cefEvent.DeviceVendor != "" {
		cefObject.Put("device.vendor", cefEvent.DeviceVendor)
	}
	if cefEvent.DeviceProduct != "" {
		cefObject.Put("device.product", cefEvent.DeviceProduct)
	}
	if cefEvent.DeviceVersion != "" {
		cefObject.Put("device.version", cefEvent.DeviceVersion)
	}
	if cefEvent.DeviceEventClassID != "" {
		cefObject.Put("device.event_class_id", cefEvent.DeviceEventClassID)
	}
	if cefEvent.Name != "" {
		cefObject.Put("name", cefEvent.Name)
	}
	if cefEvent.Severity != "" {
		cefObject.Put("severity", cefEvent.Severity)
	}

	// Add CEF extensions (key-value pairs).
	if len(cefEvent.Extensions) > 0 {
		extensions := make(common.MapStr, len(cefEvent.Extensions))
		cefObject.Put("extensions", extensions)
		for k, v := range cefEvent.Extensions {
			extensions.Put(k, v)
		}
	}

	return cefObject
}

func appendErrorMessage(m common.MapStr, msg string) error {
	const field = "error.message"
	list, _ := m.GetValue(field)

	switch v := list.(type) {
	case nil:
		m.Put(field, msg)
	case string:
		if msg != v {
			m.Put(field, []string{v, msg})
		}
	case []string:
		for _, existingTag := range v {
			if msg == existingTag {
				// Duplicate
				return nil
			}
		}
		m.Put(field, append(v, msg))
	case []interface{}:
		for _, existingTag := range v {
			if msg == existingTag {
				// Duplicate
				return nil
			}
		}
		m.Put(field, append(v, msg))
	default:
		return errors.Errorf("unexpected type %T found for %v field", list, field)
	}
	return nil
}
