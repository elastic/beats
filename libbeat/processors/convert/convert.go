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
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

const logName = "processor.convert"

var ignoredFailure = struct{}{}

func init() {
	processors.RegisterPlugin("convert", New)
	jsprocessor.RegisterPlugin("Convert", New)
}

type processor struct {
	config
	log *logp.Logger
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

	return &processor{config: c, log: log}, nil
}

func (p *processor) String() string {
	json, _ := json.Marshal(p.config)
	return "convert=" + string(json)
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	converted := make([]interface{}, len(p.Fields))

	// Convert the fields and write the results to temporary storage.
	if err := p.convertFields(event, converted); err != nil {
		return event, err
	}

	// Backup original event.
	saved := event

	if len(p.Fields) > 1 && p.FailOnError {
		// Clone the fields to allow the processor to undo the operation on
		// failure (like a transaction). If there is only one conversion then
		// cloning is unnecessary because there are no previous changes to
		// rollback (so avoid the expensive clone operation).
		saved = event.Clone()
	}

	// Update the event with the converted values.
	if err := p.writeToEvent(event, converted); err != nil {
		return saved, err
	}

	return event, nil
}

func (p *processor) convertFields(event *beat.Event, converted []interface{}) error {
	// Write conversion results to temporary storage.
	for i, conv := range p.Fields {
		v, err := p.convertField(event, conv)
		if err != nil {
			if p.FailOnError {
				return err
			}
			v = ignoredFailure
		}
		converted[i] = v
	}

	return nil
}

func (p *processor) convertField(event *beat.Event, conversion field) (interface{}, error) {
	v, err := event.GetValue(conversion.From)
	if err != nil {
		if p.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return ignoredFailure, nil
		}
		return nil, newConvertError(conversion, err, p.Tag, "field [%v] is missing", conversion.From)
	}

	if conversion.Type > unset {
		t, err := transformType(conversion.Type, v)
		if err != nil {
			return nil, newConvertError(conversion, err, p.Tag, "unable to convert value [%v]", v)
		}
		v = t
	}

	return v, nil
}

func (p *processor) writeToEvent(event *beat.Event, converted []interface{}) error {
	for i, conversion := range p.Fields {
		v := converted[i]
		if v == ignoredFailure {
			continue
		}

		if conversion.To != "" {
			switch p.Mode {
			case renameMode:
				if _, err := event.PutValue(conversion.To, v); err != nil && p.FailOnError {
					return newConvertError(conversion, err, p.Tag, "failed to put field [%v]", conversion.To)
				}
				event.Delete(conversion.From)
			case copyMode:
				if _, err := event.PutValue(conversion.To, cloneValue(v)); err != nil && p.FailOnError {
					return newConvertError(conversion, err, p.Tag, "failed to put field [%v]", conversion.To)
				}
			}
		} else {
			// In-place conversion.
			event.PutValue(conversion.From, v)
		}
	}

	return nil
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
		return strToInt(v, 64)
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
		i, err := strToInt(v, 32)
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
		return "", errors.New("value is not a valid IP address")
	default:
		return "", errors.Errorf("invalid conversion of [%T] to IP", value)
	}
}

func newConvertError(conversion field, cause error, tag string, message string, params ...interface{}) error {
	var buf strings.Builder
	buf.WriteString("failed in processor.convert")
	if tag != "" {
		buf.WriteString(" with instance_id=")
		buf.WriteString(tag)
	}
	buf.WriteString(": conversion of field [")
	buf.WriteString(conversion.From)
	buf.WriteString("] to type [")
	buf.WriteString(conversion.Type.String())
	buf.WriteString("]")
	if conversion.To != "" {
		buf.WriteString(" with target field [")
		buf.WriteString(conversion.To)
		buf.WriteString("]")
	}
	buf.WriteString(" failed: ")
	fmt.Fprintf(&buf, message, params...)
	return errors.Wrapf(cause, buf.String())
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
	case []interface{}:
		len := len(v)
		newArr := make([]interface{}, len)
		for idx, val := range v {
			newArr[idx] = cloneValue(val)
		}
		return newArr
	default:
		return value
	}
}

// strToInt is a helper to interpret a string as either base 10 or base 16.
func strToInt(s string, bitSize int) (int64, error) {
	base := 10
	if hasHexPrefix(s) {
		// strconv.ParseInt will accept the '0x' or '0X` prefix only when base is 0.
		base = 0
	}
	return strconv.ParseInt(s, base, bitSize)
}

func hasHexPrefix(s string) bool {
	if len(s) < 3 {
		return false
	}
	a, b := s[0], s[1]
	if a == '+' || a == '-' {
		a, b = b, s[2]
	}
	return a == '0' && (b == 'x' || b == 'X')
}
