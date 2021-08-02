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

package decode_xml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/encoding/xml"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

type decodeXML struct {
	decodeXMLConfig

	log *logp.Logger
}

var (
	errFieldIsNotString = errors.New("field value is not a string")
)

const (
	procName = "decode_xml"
	logName  = "processor." + procName
)

func init() {
	processors.RegisterPlugin(procName,
		checks.ConfigChecked(New,
			checks.RequireFields("field"),
			checks.AllowedFields(
				"field", "target_field",
				"overwrite_keys", "document_id",
				"to_lower", "ignore_missing",
				"ignore_failure", "when",
			)))
	jsprocessor.RegisterPlugin("DecodeXML", New)
}

// New constructs a new decode_xml processor.
func New(c *common.Config) (processors.Processor, error) {
	config := defaultConfig()

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the "+procName+" processor configuration: %s", err)
	}

	return newDecodeXML(config)
}

func newDecodeXML(config decodeXMLConfig) (processors.Processor, error) {
	// Default target to overwriting field.
	if config.Target == nil {
		config.Target = &config.Field
	}

	return &decodeXML{
		decodeXMLConfig: config,
		log:             logp.NewLogger(logName),
	}, nil
}

func (x *decodeXML) Run(event *beat.Event) (*beat.Event, error) {
	if err := x.run(event); err != nil && !x.IgnoreFailure {
		err = fmt.Errorf("failed in decode_xml on the %q field: %w", x.Field, err)
		event.PutValue("error.message", err.Error())
		return event, err
	}
	return event, nil
}

func (x *decodeXML) run(event *beat.Event) error {
	data, err := event.GetValue(x.Field)
	if err != nil {
		if x.IgnoreMissing && err == common.ErrKeyNotFound {
			return nil
		}
		return err
	}

	text, ok := data.(string)
	if !ok {
		return errFieldIsNotString
	}

	xmlOutput, err := x.decode([]byte(text))
	if err != nil {
		return fmt.Errorf("error decoding XML field: %w", err)
	}

	var id string
	if tmp, err := common.MapStr(xmlOutput).GetValue(x.DocumentID); err == nil {
		if v, ok := tmp.(string); ok {
			id = v
			common.MapStr(xmlOutput).Delete(x.DocumentID)
		}
	}

	if *x.Target != "" {
		if _, err = event.PutValue(*x.Target, xmlOutput); err != nil {
			return fmt.Errorf("failed to put value %v into field %q: %w", xmlOutput, *x.Target, err)
		}
	} else {
		jsontransform.WriteJSONKeys(event, xmlOutput, false, x.OverwriteKeys, !x.IgnoreFailure)
	}

	if id != "" {
		event.SetID(id)
	}

	return nil
}

func (x *decodeXML) decode(p []byte) (common.MapStr, error) {
	dec := xml.NewDecoder(bytes.NewReader(p))
	if x.ToLower {
		dec.LowercaseKeys()
	}

	out, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	return common.MapStr(out), nil
}

func (x *decodeXML) String() string {
	json, _ := json.Marshal(x.decodeXMLConfig)
	return procName + "=" + string(json)
}
