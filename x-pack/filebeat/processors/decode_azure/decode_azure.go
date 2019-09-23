package decode_azure

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/x-pack/filebeat/processors/decode_cef/cef"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"strconv"
	"strings"
)

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.



const (
	procName = "decode_azure"
	logName  = "processor." + procName
)

type config struct {
	Field         string `config:"field"`          // Source field containing the CEF message.
	TargetField   string `config:"target_field"`   // Target field for the CEF object.
	IgnoreMissing bool   `config:"ignore_missing"` // Ignore missing source field.
	IgnoreFailure bool   `config:"ignore_failure"` // Ignore failures when the source field does not contain a CEF message. Parse errors do not cause failures, but are added to error.message.
	ID            string `config:"id"`             // Instance ID for debugging purposes.
	ECS           bool   `config:"ecs"`            // Generate ECS fields.
}

func defaultConfig() config {
	return config{
		Field:       "message",
		TargetField: "cef",
		ECS:         true,
	}
}

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
	cfgwarn.Beta("The " + procName + " processor is a beta feature.")

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
	if err = ce.Unpack([]byte(cefData), cef.WithFullExtensionNames()); ce.Version < 0 && err != nil {
		if p.IgnoreFailure {
			return event, nil
		}
		return event, errors.Wrap(err, "decode_cef failed to parse message")
	}

	cefErrors := multierr.Errors(err)
	cefObject := toCEFObject(&ce)
	event.PutValue(p.TargetField, cefObject)


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
	cefObject := common.MapStr{"version": strconv.Itoa(cefEvent.Version)}
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


