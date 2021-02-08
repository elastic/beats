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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/enc/mxj"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

type decodeXML struct {
	config decodeXMLConfig
	log    *logp.Logger
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
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "overwrite_keys", "add_error_key", "target", "document_id")))
	jsprocessor.RegisterPlugin(procName, New)
}

// New construct a new decode_xml processor.
func New(c *common.Config) (processors.Processor, error) {
	config := defaultConfig()

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the "+procName+" processor configuration: %s", err)
	}

	return newDecodeXML(config)

}

func newDecodeXML(config decodeXMLConfig) (processors.Processor, error) {
	cfgwarn.Experimental("The " + procName + " processor is experimental.")

	log := logp.NewLogger(logName)

	return &decodeXML{config: config, log: log}, nil

}

func (x *decodeXML) Run(event *beat.Event) (*beat.Event, error) {
	var errs []error
	var field = x.config.Field
	data, err := event.GetValue(field)
	if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
		errs = append(errs, err)
	}
	text, ok := data.(string)
	if !ok {
		errs = append(errs, errFieldIsNotString)
	}
	xmloutput, err := x.decodeField(text)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to decode fields in decode_xml processor: %v", err))
	}

	target := field
	if x.config.Target != "" {
		target = x.config.Target
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
		errs = append(errs, fmt.Errorf("Error trying to Put value %v for field: %s. Error: %w", xmloutput, field, err))
	}
	if id != "" {
		event.SetID(id)
	}
	// If error has not already been set, add errors if ignore_failure is false.
	if len(errs) > 0 {
		var combinedErrors = multierr.Combine(errs...)
		if x.config.AddErrorKey {
			event.Fields["error"] = combinedErrors.Error()
		}
		return event, combinedErrors
	}
	return event, nil
}

func (x *decodeXML) decodeField(data string) (decodedData map[string]interface{}, err error) {
	decodedData, err = mxj.UnmarshalXML([]byte(data), false, x.config.ToLower)
	if err != nil {
		return nil, fmt.Errorf("error decoding XML field: %w", err)
	}

	return decodedData, nil
}

func (x *decodeXML) String() string {
	json, _ := json.Marshal(x.config)
	return procName + "=" + string(json)
}
