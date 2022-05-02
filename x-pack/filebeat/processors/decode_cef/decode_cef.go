// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/x-pack/filebeat/processors/decode_cef/cef"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
func New(cfg *conf.C) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the "+procName+" processor configuration")
	}

	return newDecodeCEF(c)
}

func newDecodeCEF(c config) (*processor, error) {
	log := logp.NewLogger(logName)
	if c.ID != "" {
		log = log.With("instance_id", c.ID)
	}

	return &processor{config: c, log: log}, nil
}

func (p *processor) String() string {
	json, _ := json.Marshal(p.config)
	return procName + "=" + string(json)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	v, err := event.GetValue(p.Field)
	if err != nil {
		if p.IgnoreMissing {
			return event, nil
		}
		return event, errors.Wrapf(err, "decode_cef field [%v] not found", p.Field)
	}

	cefData, ok := v.(string)
	if !ok {
		if p.IgnoreFailure {
			return event, nil
		}
		return event, errors.Wrapf(err, "decode_cef field [%v] is not a string", p.Field)
	}

	// Ignore any leading data before the CEF header.
	idx := strings.Index(cefData, "CEF:")
	if idx == -1 {
		if p.IgnoreFailure {
			return event, nil
		}
		return event, errors.Errorf("decode_cef field [%v] does not contain a CEF header", p.Field)
	}
	cefData = cefData[idx:]

	// If the version < 0 after parsing then none of the data is valid so return here.
	var ce cef.Event
	if err = ce.Unpack(cefData, cef.WithFullExtensionNames(), cef.WithTimezone(p.Timezone.Location())); ce.Version < 0 && err != nil {
		if p.IgnoreFailure {
			return event, nil
		}
		return event, errors.Wrap(err, "decode_cef failed to parse message")
	}

	cefErrors := multierr.Errors(err)
	cefObject := toCEFObject(&ce)
	event.PutValue(p.TargetField, cefObject)

	// Map CEF extension fields to ECS fields.
	if p.ECS {
		writeCEFHeaderToECS(&ce, event)

		for key, field := range ce.Extensions {
			mapping, found := ecsExtensionMapping[key]
			if !found {
				continue
			}

			// Apply translation function or use a standard type translation (e.g. string to long).
			if mapping.Translate != nil {
				translatedValue, err := mapping.Translate(field)
				if err != nil {
					cefErrors = append(cefErrors, errors.Wrap(err, key))
					continue
				}
				if translatedValue != nil {
					event.PutValue(mapping.Target, translatedValue)
				}
			} else if field.Interface != nil {
				event.PutValue(mapping.Target, field.Interface)
			} else {
				event.PutValue(mapping.Target, field.String)
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

func toCEFObject(cefEvent *cef.Event) mapstr.M {
	// Add CEF header fields.
	cefObject := mapstr.M{"version": strconv.Itoa(cefEvent.Version)}
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
		extensions := make(mapstr.M, len(cefEvent.Extensions))
		cefObject.Put("extensions", extensions)
		for k, field := range cefEvent.Extensions {
			if field.Interface != nil {
				extensions.Put(k, field.Interface)
			} else {
				extensions.Put(k, field.String)
			}
		}
	}

	return cefObject
}

func writeCEFHeaderToECS(cefEvent *cef.Event, event *beat.Event) {
	if cefEvent.DeviceVendor != "" {
		event.PutValue("observer.vendor", cefEvent.DeviceVendor)
	}
	if cefEvent.DeviceProduct != "" {
		// TODO: observer.product is not officially part of ECS.
		event.PutValue("observer.product", cefEvent.DeviceProduct)
	}
	if cefEvent.DeviceVersion != "" {
		event.PutValue("observer.version", cefEvent.DeviceVersion)
	}
	if cefEvent.DeviceEventClassID != "" {
		event.PutValue("event.code", cefEvent.DeviceEventClassID)
	}
	if cefEvent.Name != "" {
		event.PutValue("message", cefEvent.Name)
	}
	if cefEvent.Severity != "" {
		if sev, ok := cefSeverityToNumber(cefEvent.Severity); ok {
			event.PutValue("event.severity", sev)
		}
	}
}

func appendErrorMessage(m mapstr.M, msg string) error {
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

// cefSeverityToNumber converts the CEF severity string to a numeric value. The
// returned boolean indicates if the conversion was successful.
func cefSeverityToNumber(severity string) (int, bool) {
	// From CEF spec:
	// Severity is a string or integer and reflects the importance of the event.
	// The valid string values are Unknown, Low, Medium, High, and Very-High.
	// The valid integer values are 0-3=Low, 4-6=Medium, 7- 8=High, and 9-10=Very-High.
	switch strings.ToLower(severity) {
	case "low":
		return 0, true
	case "medium":
		return 4, true
	case "high":
		return 7, true
	case "very-high":
		return 9, true
	default:
		s, err := strconv.Atoi(severity)
		return s, err == nil
	}
}
