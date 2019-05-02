// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package convert

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const logName = "processor.convert"

func init() {
	processors.RegisterPlugin("convert", New)
}

type processor struct {
	config
	log *logp.Logger

	converted []interface{} // Temporary storage for converted values.
}

// New constructs a new convert processor.
func New(cfg *common.Config) (processors.Processor, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, errors.Wrap(err, "fail to unpack the convert processor configuration")
	}

	return newConvert(c)
}

func newConvert(c config) (*processor, error) {
	log := logp.NewLogger(logName)
	if c.Tag != "" {
		log = log.With("instance_id", c.Tag)
	}

	return &processor{config: c, log: log, converted: make([]interface{}, len(c.Fields))}, nil
}

func (p *processor) String() string {
	json, _ := json.Marshal(p.config)
	return "convert=" + string(json)
}

var ignoredFailure = struct{}{}

func resetValues(s []interface{}) {
	for i := range s {
		s[i] = nil
	}
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	defer resetValues(p.converted)

	// Validate the conversions.
	for i, conv := range p.Fields {
		v, err := convertField(event, conv.From, conv.Type)
		if err != nil {
			switch cause := errors.Cause(err); cause {
			case common.ErrKeyNotFound:
				if !p.IgnoreMissing && p.FailOnError {
					return event, annotateError(p.Tag, errors.Errorf("field [%v] is missing, cannot be converted to type [%v]", conv.From, conv.Type))
				}
			default:
				if p.FailOnError {
					return event, annotateError(p.Tag, errors.Wrapf(err, "unable to convert field [%v] value [%v] to [%v]", conv.From, v, conv.Type))
				}
			}
			p.converted[i] = ignoredFailure
			continue
		}
		p.converted[i] = v
	}

	saved := *event
	if len(p.Fields) > 1 && p.FailOnError {
		// Clone the fields to allow the processor to undo the operation on
		// failure (like a transaction). If there is only one conversion then
		// cloning is unnecessary because there are no previous changes to
		// rollback (so avoid the expensive clone operation).
		saved.Fields = event.Fields.Clone()
		saved.Meta = event.Meta.Clone()
	}

	for i, conv := range p.Fields {
		v := p.converted[i]
		if v == ignoredFailure {
			continue
		}

		if conv.To != "" {
			switch p.Mode {
			case renameMode:
				if _, err := event.PutValue(conv.To, v); err != nil && p.FailOnError {
					return &saved, annotateError(p.Tag, errors.Wrapf(err, "failed to put field [%v]", conv.To))
				}
				event.Delete(conv.From)
			case copyMode:
				if _, err := event.PutValue(conv.To, cloneValue(v)); err != nil && p.FailOnError {
					return &saved, annotateError(p.Tag, errors.Wrapf(err, "failed to put field [%v]", conv.To))
				}
			}
		} else {
			// In-place conversion.
			event.PutValue(conv.From, v)
		}
	}

	return event, nil
}

func convertField(event *beat.Event, from string, typ dataType) (interface{}, error) {
	v, err := event.GetValue(from)
	if err != nil {
		return nil, err
	}

	if typ > unset {
		v, err = transformType(typ, v)
		if err != nil {
			return nil, err
		}
	}

	return v, nil
}

func transformType(typ dataType, value interface{}) (interface{}, error) {
	switch typ {
	case String:
		return toString(value)
	case Long:
		return toLong(value)
	case Integer:
		return toInteger(value)
	case Float:
		return toFloat(value)
	case Double:
		return toDouble(value)
	case Boolean:
		return toBoolean(value)
	case IP:
		return toIP(value)
	default:
		return value, nil
	}
}

func toString(value interface{}) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", errors.New("invalid conversion of [null] to string")
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

func toLong(value interface{}) (int64, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseInt(v, 0, 64)
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, errors.Errorf("invalid conversion of [%T] to long", value)
	}
}

func toInteger(value interface{}) (int32, error) {
	switch v := value.(type) {
	case string:
		i, err := strconv.ParseInt(v, 0, 32)
		return int32(i), err
	case int:
		return int32(v), nil
	case int8:
		return int32(v), nil
	case int16:
		return int32(v), nil
	case int32:
		return v, nil
	case int64:
		return int32(v), nil
	case uint:
		return int32(v), nil
	case uint8:
		return int32(v), nil
	case uint16:
		return int32(v), nil
	case uint32:
		return int32(v), nil
	case uint64:
		return int32(v), nil
	case float32:
		return int32(v), nil
	case float64:
		return int32(v), nil
	default:
		return 0, errors.Errorf("invalid conversion of [%T] to integer", value)
	}
}

func toFloat(value interface{}) (float32, error) {
	switch v := value.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 32)
		return float32(f), err
	case int:
		return float32(v), nil
	case int8:
		return float32(v), nil
	case int16:
		return float32(v), nil
	case int32:
		return float32(v), nil
	case int64:
		return float32(v), nil
	case uint:
		return float32(v), nil
	case uint8:
		return float32(v), nil
	case uint16:
		return float32(v), nil
	case uint32:
		return float32(v), nil
	case uint64:
		return float32(v), nil
	case float32:
		return v, nil
	case float64:
		return float32(v), nil
	default:
		return 0, errors.Errorf("invalid conversion of [%T] to float", value)
	}
}

func toDouble(value interface{}) (float64, error) {
	switch v := value.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return float64(f), err
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, errors.Errorf("invalid conversion of [%T] to float", value)
	}
}

func toBoolean(value interface{}) (bool, error) {
	switch v := value.(type) {
	case string:
		return strconv.ParseBool(v)
	case bool:
		return v, nil
	default:
		return false, errors.Errorf("invalid conversion of [%T] to boolean", value)
	}
}

func toIP(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		// This is validating that the value is an IP.
		if net.ParseIP(v) != nil {
			return v, nil
		}
	}
	return "", errors.Errorf("invalid conversion of [%T] to IP", value)
}

func annotateError(id string, err error) error {
	if err == nil {
		return nil
	}
	if id != "" {
		return errors.Wrapf(err, "failed in processor.convert with instance_id=%v", id)
	}
	return errors.Wrap(err, "failed in processor.convert")
}

// cloneValue returns a shallow copy of a map. All other types are passed
// through in the return. This should be used when making straight copies of
// maps without doing any type conversions.
func cloneValue(value interface{}) interface{} {
	switch v := value.(type) {
	case common.MapStr:
		return v.Clone()
	case map[string]interface{}:
		return common.MapStr(v).Clone()
	default:
		return value
	}
}
