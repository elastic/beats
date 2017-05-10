package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/jsontransform"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/pkg/errors"
)

type decodeJSONFields struct {
	fields        []string
	maxDepth      int
	overwriteKeys bool
	processArray  bool
	target        *string
}

type config struct {
	Fields        []string `config:"fields"`
	MaxDepth      int      `config:"max_depth" validate:"min=1"`
	OverwriteKeys bool     `config:"overwrite_keys"`
	ProcessArray  bool     `config:"process_array"`
	Target        *string  `config:"target"`
}

var (
	defaultConfig = config{
		MaxDepth:     1,
		ProcessArray: false,
	}
)

var debug = logp.MakeDebug("filters")

func init() {
	processors.RegisterPlugin("decode_json_fields",
		configChecked(newDecodeJSONFields,
			requireFields("fields"),
			allowedFields("fields", "max_depth", "overwrite_keys", "process_array", "target", "when")))
}

func newDecodeJSONFields(c common.Config) (processors.Processor, error) {
	config := defaultConfig

	err := c.Unpack(&config)

	if err != nil {
		logp.Warn("Error unpacking config for decode_json_fields")
		return nil, fmt.Errorf("fail to unpack the decode_json_fields configuration: %s", err)
	}

	f := decodeJSONFields{fields: config.Fields, maxDepth: config.MaxDepth, overwriteKeys: config.OverwriteKeys, processArray: config.ProcessArray, target: config.Target}
	return f, nil
}

func (f decodeJSONFields) Run(event common.MapStr) (common.MapStr, error) {
	var errs []string

	for _, field := range f.fields {
		data, err := event.GetValue(field)
		if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
			debug("Error trying to GetValue for field : %s in event : %v", field, event)
			errs = append(errs, err.Error())
			continue
		}
		text, ok := data.(string)
		if ok {
			var output interface{}
			err := unmarshal(f.maxDepth, []byte(text), &output, f.processArray)
			if err != nil {
				debug("Error trying to unmarshal %s", event[field])
				errs = append(errs, err.Error())
				continue
			}

			if f.target != nil {
				if len(*f.target) > 0 {
					_, err = event.Put(*f.target, output)
				} else {
					switch t := output.(type) {
					default:
						errs = append(errs, errors.New("Error trying to add target to root.").Error())
					case map[string]interface{}:
						jsontransform.WriteJSONKeys(event, t, f.overwriteKeys)
					}
				}
			} else {
				_, err = event.Put(field, output)
			}

			if err != nil {
				debug("Error trying to Put value %v for field : %s", output, field)
				errs = append(errs, err.Error())
				continue
			}
		}
	}

	if len(errs) > 0 {
		return event, fmt.Errorf(strings.Join(errs, ", "))
	}
	return event, nil
}

func unmarshal(maxDepth int, text []byte, fields *interface{}, processArray bool) error {
	if err := DecodeJSON(text, fields); err != nil {
		return err
	}

	maxDepth--
	if maxDepth == 0 {
		return nil
	}

	tryUnmarshal := func(v interface{}) (interface{}, bool) {
		str, isString := v.(string)
		if !isString {
			return v, false
		}

		var tmp interface{}
		err := unmarshal(maxDepth, []byte(str), &tmp, processArray)
		if err != nil {
			return v, false
		}

		return tmp, true
	}

	// try to deep unmarshal fields
	switch O := interface{}(*fields).(type) {
	case map[string]interface{}:
		for k, v := range O {
			if decoded, ok := tryUnmarshal(v); ok {
				O[k] = decoded
			}
		}
	// We want to process arrays here
	case []interface{}:
		if !processArray {
			break
		}

		for i, v := range O {
			if decoded, ok := tryUnmarshal(v); ok {
				O[i] = decoded
			}
		}
	}
	return nil
}

func DecodeJSON(text []byte, to *interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	err := dec.Decode(to)

	if err != nil {
		return err
	}

	if dec.More() {
		return errors.New("multiple json elements found")
	}

	switch O := interface{}(*to).(type) {
	case map[string]interface{}:
		jsontransform.TransformNumbers(O)
	}
	return nil
}

func (f decodeJSONFields) String() string {
	return "decode_json_fields=" + strings.Join(f.fields, ", ")
}
