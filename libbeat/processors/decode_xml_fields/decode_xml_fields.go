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

package decode_xml_fields

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

type decodeXMLFieldsConfig struct {
	Fields        []string `config:"fields"`
	OverwriteKeys bool     `config:"overwrite_keys"`
	AddErrorKey   bool     `config:"add_error_key"`
	Target        *string  `config:"target"`
	DocumentID    string   `config:"document_id"`
	ToLower       bool     `config:"to_lower"`
}

type decodeXMLFields struct {
	config decodeXMLFieldsConfig
	logger *logp.Logger
}

var (
	defaultConfig = decodeXMLFieldsConfig{
		Fields:        []string{"message"},
		OverwriteKeys: false,
		AddErrorKey:   false,
		ToLower:       true,
	}
	errFieldIsNotString string = "The configured field is not a string"
)

func init() {
	processors.RegisterPlugin("decode_xml_fields",
		checks.ConfigChecked(NewDecodeXMLFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "overwrite_keys", "add_error_key", "target", "document_id")))
	jsprocessor.RegisterPlugin("decode_xml_fields", NewDecodeXMLFields)
}

// NewDecodeXMLFields construct a new decode_xml_fields processor.
func NewDecodeXMLFields(c *common.Config) (processors.Processor, error) {
	config := defaultConfig

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the decode_xml_fields configuration: %s", err)
	}

	return &decodeXMLFields{
		config: config,
		logger: logp.NewLogger("decode_xml_fields"),
	}, nil

}

func (x *decodeXMLFields) Run(event *beat.Event) (*beat.Event, error) {
	var errs []string

	for _, field := range x.config.Fields {
		data, err := event.GetValue(field)
		if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
			x.logger.Debugf("Error trying to GetValue for field : %s in event : %v", field, event)
			errs = append(errs, err.Error())
			continue
		}
		text, ok := data.(string)
		if !ok {
			errs = append(errs, errFieldIsNotString)
			continue
		}
		xmloutput, err := x.decodeField(field, text)
		if err != nil {
			errs = append(errs, err.Error())
			x.logger.Errorf("failed to decode fields in decode_xml_fields processor: %v", err)
		}

		target := field
		if x.config.Target != nil {
			target = *x.config.Target
		}

		var id string
		if key := x.config.DocumentID; key != "" {
			if tmp, err := common.MapStr(xmloutput).GetValue(key); err == nil {
				if v, ok := tmp.(string); ok {
					id = v
					common.MapStr(xmloutput).Delete(key)
				}
			}
		}

		if target != "" {
			_, err = event.PutValue(target, xmloutput)
		} else {
			jsontransform.WriteJSONKeys(event, xmloutput, false, x.config.OverwriteKeys, x.config.AddErrorKey)
		}

		if err != nil {
			x.logger.Debugf("Error trying to Put value %v for field : %s", xmloutput, field)
			errs = append(errs, err.Error())
			continue
		}
		if id != "" {
			if event.Meta == nil {
				event.Meta = common.MapStr{}
			}
			event.Meta[events.FieldMetaID] = id
		}
	}
	// If jsontransform has not set an error, and it happened elsewhere, add it to the error
	if len(errs) > 0 {
		if event.Fields["error"] == nil {
			setError(event, errs, x.config.AddErrorKey)
		}
		return event, fmt.Errorf(strings.Join(errs, ", "))
	}
	return event, nil
}

func (x *decodeXMLFields) decodeField(field string, data string) (decodedData map[string]interface{}, err error) {
	decodedData, err = common.UnmarshalXML([]byte(data), false, x.config.ToLower)
	if err != nil {
		return nil, fmt.Errorf("error trying to decode XML field %v", err)
	}

	return decodedData, nil
}

func (x *decodeXMLFields) String() string {
	return "decode_xml_fields=" + fmt.Sprintf("%+v", x.config.Fields)
}

func setError(event *beat.Event, errs []string, addErrKey bool) {
	if addErrKey {
		event.Fields["error"] = errs
	}
}
