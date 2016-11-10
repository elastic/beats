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
	Fields []string
}

var debug = logp.MakeDebug("filters")

func init() {
	processors.RegisterPlugin("decode_json_fields", configChecked(newDecodeJSONFields,
		requireFields("fields"), allowedFields("fields", "when")))
}

func newDecodeJSONFields(c common.Config) (processors.Processor, error) {
	config := struct {
		Fields []string `config:"fields"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		logp.Warn("Error unpacking config for decode_json_fields")
		return nil, fmt.Errorf("fail to unpack the decode_json_fields configuration: %s", err)
	}

	f := decodeJSONFields{Fields: config.Fields}
	return f, nil
}

func (f decodeJSONFields) Run(event common.MapStr) (common.MapStr, error) {
	var errs []string

	for _, field := range f.Fields {
		data, err := event.GetValue(field)
		if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
			debug("Error trying to GetValue for field : %s in event : %v", field, event)
			errs = append(errs, err.Error())
			continue
		}
		text, ok := data.(string)
		if ok {
			var output map[string]interface{}
			err := unmarshal([]byte(text), &output)
			if err != nil {
				debug("Error trying to unmarshal %s", event[field])
				errs = append(errs, err.Error())
				continue
			}

			_, err = event.Put(field, output)
			if err != nil {
				debug("Error trying to Put value %v for field : %s", output, field)
				errs = append(errs, err.Error())
				continue
			}
		}
	}

	return event, fmt.Errorf(strings.Join(errs, ", "))
}

// unmarshal is equivalent with json.Unmarshal but it converts numbers
// to int64 where possible, instead of using always float64.
func unmarshal(text []byte, fields *map[string]interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	err := dec.Decode(fields)
	if err != nil {
		return err
	}

	//Iterate through all the fields to perform deep parsing
	for k, v := range *fields {
		switch vv := v.(type) {
		case string:
			var output map[string]interface{}
			sErr := unmarshal([]byte(vv), &output)
			if sErr == nil {
				(*fields)[k] = output
			}
		}
	}

	jsontransform.TransformNumbers(*fields)
	return nil
}

func (f decodeJSONFields) String() string {
	return "decode_json_fields=" + strings.Join(f.Fields, ", ")
}
