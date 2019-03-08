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

package mapping

import (
	"io/ioutil"

	"github.com/elastic/beats/libbeat/asset"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Supporter returns the configured fields bytes
type Supporter interface {
	GetBytes() ([]byte, error)
}

// fieldsSupport returns the default fields of a Beat
type fieldsSupport struct {
	log  *logp.Logger
	beat string
}

// customFieldsSupport returns the custom configured fields of a Beat
type customFieldsSupport struct {
	log  *logp.Logger
	beat string
	path string
}

// appendFieldsSupport return the default fields of a Beat
// and the additional fields from append_fields
type appendFieldsSupport struct {
	original       Supporter
	appendedFields []byte
}

// externalFieldsSupport return the default fields of a Beat
// and the additional fields from external_fields
type externalFieldsSupport struct {
	original Supporter
	paths    []string
}

func DefaultSupport(log *logp.Logger, beat string, cfg *common.Config) (Supporter, error) {
	if log == nil {
		log = logp.NewLogger("mapping")
	} else {
		log = log.Named("mapping")
	}

	config := struct {
		Template struct {
			Fields         string   `config:"fields"`
			AppendFields   []byte   `config:"append_fields"`
			ExternalFields []string `config:"external_fields"`
		} `config:"setup.template"`
	}{}
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// load default fields.ymls of Beat
	var s Supporter
	s = &fieldsSupport{
		log:  log,
		beat: beat,
	}

	// custom fields.yml file is configured
	if config.Template.Fields != "" {
		s = &customFieldsSupport{
			log:  log,
			beat: beat,
			path: config.Template.Fields,
		}
	}

	// append_fields
	if len(config.Template.AppendFields) > 0 {
		s = &appendFieldsSupport{
			original:       s,
			appendedFields: config.Template.AppendFields,
		}
	}

	// external_fields
	if len(config.Template.ExternalFields) > 0 {
		s = &externalFieldsSupport{
			original: s,
			paths:    config.Template.ExternalFields,
		}
	}

	return s, nil
}

func (f *fieldsSupport) GetBytes() ([]byte, error) {
	return asset.GetFields(f.beat)
}

func (c *customFieldsSupport) GetBytes() ([]byte, error) {
	c.log.Debugf("Reading bytes custom fields.yml from %s", c.path)
	return ioutil.ReadFile(c.path)
}

func (a *appendFieldsSupport) GetBytes() ([]byte, error) {
	fields, err := a.original.GetBytes()
	if err != nil {
		return nil, err
	}

	return append(fields, a.appendedFields...), nil
}

func (e *externalFieldsSupport) GetBytes() ([]byte, error) {
	fields, err := e.original.GetBytes()
	if err != nil {
		return nil, err
	}
	for _, path := range e.paths {
		f, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f...)
	}

	return fields, nil
}
